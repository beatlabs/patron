// Package sqs provides a native consumer for AWS SQS.
package sqs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/observability/log"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultStatsInterval = 10 * time.Second
	defaultRetries       = 10
	defaultRetryWait     = time.Second
	defaultMaxMessages   = 3
)

// ProcessorFunc definition of an async processor.
type ProcessorFunc func(context.Context, Batch)

type messageState string

const (
	sqsAttributeApproximateNumberOfMessages           = "ApproximateNumberOfMessages"
	sqsAttributeApproximateNumberOfMessagesDelayed    = "ApproximateNumberOfMessagesDelayed"
	sqsAttributeApproximateNumberOfMessagesNotVisible = "ApproximateNumberOfMessagesNotVisible"
	sqsAttributeSentTimestamp                         = "SentTimestamp"

	sqsMessageAttributeAll = "All"

	consumerComponent = "sqs-consumer"

	ackMessageState     messageState = "ACK"
	nackMessageState    messageState = "NACK"
	fetchedMessageState messageState = "FETCHED"
)

type retry struct {
	count uint
	wait  time.Duration
}

type config struct {
	maxMessages       int32
	pollWaitSeconds   int32
	visibilityTimeout int32
}

type stats struct {
	interval time.Duration
}

type API interface {
	CreateQueue(ctx context.Context, params *sqs.CreateQueueInput, optFns ...func(*sqs.Options)) (*sqs.CreateQueueOutput, error)
	GetQueueUrl(ctx context.Context, params *sqs.GetQueueUrlInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
	DeleteMessageBatch(ctx context.Context, params *sqs.DeleteMessageBatchInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageBatchOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
}

// Component implementation of an async component.
type Component struct {
	name       string
	queue      queue
	queueOwner string
	api        API
	cfg        config
	proc       ProcessorFunc
	stats      stats
	retry      retry
}

// New creates a new component with support for functional configuration.
func New(name, queueName string, sqsAPI API, proc ProcessorFunc, oo ...OptionFunc) (*Component, error) {
	if name == "" {
		return nil, errors.New("component name is empty")
	}

	if queueName == "" {
		return nil, errors.New("queue name is empty")
	}

	if sqsAPI == nil {
		return nil, errors.New("SQS API is nil")
	}

	if proc == nil {
		return nil, errors.New("process function is nil")
	}

	cmp := &Component{
		name: name,
		queue: queue{
			name: queueName,
		},
		api: sqsAPI,
		cfg: config{
			maxMessages: defaultMaxMessages,
		},
		stats: stats{interval: defaultStatsInterval},
		proc:  proc,
		retry: retry{
			count: defaultRetries,
			wait:  defaultRetryWait,
		},
	}

	for _, optionFunc := range oo {
		err := optionFunc(cmp)
		if err != nil {
			return nil, err
		}
	}

	req := &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	}

	if cmp.queueOwner != "" {
		req.QueueOwnerAWSAccountId = aws.String(cmp.queueOwner)
	}

	out, err := sqsAPI.GetQueueUrl(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue URL: %w", err)
	}

	cmp.queue.url = aws.ToString(out.QueueUrl)

	return cmp, nil
}

// Run starts the consumer processing loop messages.
func (c *Component) Run(ctx context.Context) error {
	chErr := make(chan error)

	go c.consume(ctx, chErr)

	tickerStats := time.NewTicker(c.stats.interval)
	defer tickerStats.Stop()
	for {
		select {
		case err := <-chErr:
			return err
		case <-ctx.Done():
			log.FromContext(ctx).Info("context cancellation received. exiting...")
			return nil
		case <-tickerStats.C:
			err := c.report(ctx, c.api, c.queue.url)
			if err != nil {
				log.FromContext(ctx).Error("failed to report sqsAPI stats", log.ErrorAttr(err))
			}
		}
	}
}

func (c *Component) consume(ctx context.Context, chErr chan error) {
	logger := log.FromContext(ctx)

	retries := c.retry.count

	for {
		if ctx.Err() != nil {
			return
		}
		logger.Debug("consume: polling SQS sqsAPI", slog.String("name", c.queue.name), slog.Int("max", int(c.cfg.maxMessages)))
		output, err := c.api.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &c.queue.url,
			MaxNumberOfMessages: c.cfg.maxMessages,
			WaitTimeSeconds:     c.cfg.pollWaitSeconds,
			VisibilityTimeout:   c.cfg.visibilityTimeout,
			MessageSystemAttributeNames: []types.MessageSystemAttributeName{
				sqsAttributeSentTimestamp,
			},
			MessageAttributeNames: []string{
				sqsMessageAttributeAll,
			},
		})
		if err != nil {
			logger.Error("failed to receive messages, sleeping", log.ErrorAttr(err), slog.Duration("wait", c.retry.wait))
			time.Sleep(c.retry.wait)
			retries--
			if retries > 0 {
				continue
			}
			chErr <- err
			return
		}
		retries = c.retry.count

		if ctx.Err() != nil {
			return
		}

		logger.Debug("consume: received messages", slog.Int("count", len(output.Messages)))
		observeMessageCount(ctx, c.queue.name, fetchedMessageState, nil, len(output.Messages))

		if len(output.Messages) == 0 {
			continue
		}

		btc := c.createBatch(ctx, output)

		c.proc(ctx, btc)
	}
}

func (c *Component) createBatch(ctx context.Context, output *sqs.ReceiveMessageOutput) batch {
	btc := batch{
		ctx:      ctx,
		queue:    c.queue,
		sqsAPI:   c.api,
		messages: make([]Message, 0, len(output.Messages)),
	}

	for _, msg := range output.Messages {
		observerMessageAge(ctx, c.queue.name, msg.Attributes)

		corID := getCorrelationID(msg.MessageAttributes)

		ctx = otel.GetTextMapPropagator().Extract(ctx, &consumerMessageCarrier{msg: &msg}) // nolint:gosec

		ctx, sp := patrontrace.StartSpan(ctx, consumerComponent, trace.WithSpanKind(trace.SpanKindConsumer))

		ctx = correlation.ContextWithID(ctx, corID)
		ctx = log.WithContext(ctx, slog.With(slog.String(correlation.ID, corID)))

		btc.messages = append(btc.messages, message{
			ctx:   ctx,
			queue: c.queue,
			api:   c.api,
			msg:   msg,
			span:  sp,
		})
	}

	return btc
}

func (c *Component) report(ctx context.Context, sqsAPI API, queueURL string) error {
	slog.Debug("retrieve stats for SQS", slog.String("queue", c.queue.name))
	rsp, err := sqsAPI.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		AttributeNames: []types.QueueAttributeName{
			sqsAttributeApproximateNumberOfMessages,
			sqsAttributeApproximateNumberOfMessagesDelayed,
			sqsAttributeApproximateNumberOfMessagesNotVisible,
		},
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		return err
	}

	size, err := getAttributeFloat64(rsp.Attributes, sqsAttributeApproximateNumberOfMessages)
	if err != nil {
		return err
	}
	observeQueueSize(ctx, c.queue.name, "available", size)

	size, err = getAttributeFloat64(rsp.Attributes, sqsAttributeApproximateNumberOfMessagesDelayed)
	if err != nil {
		return err
	}
	observeQueueSize(ctx, c.queue.name, "delayed", size)

	size, err = getAttributeFloat64(rsp.Attributes, sqsAttributeApproximateNumberOfMessagesNotVisible)
	if err != nil {
		return err
	}
	observeQueueSize(ctx, c.queue.name, "invisible", size)
	return nil
}

func getAttributeFloat64(attr map[string]string, key string) (float64, error) {
	valueString := attr[key]
	if len(strings.TrimSpace(valueString)) == 0 {
		return 0.0, fmt.Errorf("value of %s is empty", key)
	}
	value, err := strconv.ParseFloat(valueString, 64)
	if err != nil {
		return 0.0, fmt.Errorf("could not convert %s to float64", valueString)
	}
	return value, nil
}

func getCorrelationID(ma map[string]types.MessageAttributeValue) string {
	for key, value := range ma {
		if key == correlation.HeaderID {
			if value.StringValue != nil {
				return *value.StringValue
			}
			break
		}
	}
	return uuid.New().String()
}

type consumerMessageCarrier struct {
	msg *types.Message
}

// Get retrieves a single value for a given key.
func (c consumerMessageCarrier) Get(key string) string {
	return c.msg.Attributes[key]
}

// Set sets a header.
func (c consumerMessageCarrier) Set(key, val string) {
	c.msg.Attributes[key] = val
}

// Keys returns a slice of all key identifiers in the carrier.
func (c consumerMessageCarrier) Keys() []string {
	return nil
}
