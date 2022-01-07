// Package sqs provides a native consumer for AWS SQS.
package sqs

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	defaultStatsInterval = 10 * time.Second
	defaultRetries       = 10
	defaultRetryWait     = time.Second
	defaultMaxMessages   = 3
)

// ProcessorFunc definition of a async processor.
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

var (
	messageAge        *prometheus.GaugeVec
	messageCounterVec *prometheus.CounterVec
	queueSize         *prometheus.GaugeVec
)

func init() {
	messageAge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "component",
			Subsystem: "sqs",
			Name:      "message_age",
			Help:      "Message age based on the SentTimestamp SQS attribute",
		},
		[]string{"queue"},
	)
	prometheus.MustRegister(messageAge)
	messageCounterVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "sqs",
			Name:      "message_counter",
			Help:      "Message counter",
		},
		[]string{"queue", "state", "hasError"},
	)
	prometheus.MustRegister(messageCounterVec)
	queueSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "component",
			Subsystem: "sqs",
			Name:      "queue_size",
			Help:      "Queue size reported by AWS",
		},
		[]string{"state"},
	)
	prometheus.MustRegister(queueSize)
}

type retry struct {
	count uint
	wait  time.Duration
}

type config struct {
	maxMessages       *int64
	pollWaitSeconds   *int64
	visibilityTimeout *int64
}

type stats struct {
	interval time.Duration
}

// Component implementation of a async component.
type Component struct {
	name  string
	queue queue
	api   sqsiface.SQSAPI
	cfg   config
	proc  ProcessorFunc
	stats stats
	retry retry
}

// New creates a new component with support for functional configuration.
func New(name, queueName string, sqsAPI sqsiface.SQSAPI, proc ProcessorFunc, oo ...OptionFunc) (*Component, error) {
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

	out, err := sqsAPI.GetQueueUrlWithContext(context.Background(), &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get queue URL: %w", err)
	}

	cmp := &Component{
		name: name,
		queue: queue{
			name: queueName,
			url:  aws.StringValue(out.QueueUrl),
		},
		api: sqsAPI,
		cfg: config{
			maxMessages:       aws.Int64(defaultMaxMessages),
			pollWaitSeconds:   nil,
			visibilityTimeout: nil,
		},
		stats: stats{interval: defaultStatsInterval},
		proc:  proc,
		retry: retry{
			count: defaultRetries,
			wait:  defaultRetryWait,
		},
	}

	for _, optionFunc := range oo {
		err = optionFunc(cmp)
		if err != nil {
			return nil, err
		}
	}

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
				log.FromContext(ctx).Errorf("failed to report sqsAPI stats: %v", err)
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
		logger.Debugf("consume: polling SQS sqsAPI %s for %d messages", c.queue.name, *c.cfg.maxMessages)
		output, err := c.api.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &c.queue.url,
			MaxNumberOfMessages: c.cfg.maxMessages,
			WaitTimeSeconds:     c.cfg.pollWaitSeconds,
			VisibilityTimeout:   c.cfg.visibilityTimeout,
			AttributeNames: aws.StringSlice([]string{
				sqsAttributeSentTimestamp,
			}),
			MessageAttributeNames: aws.StringSlice([]string{
				sqsMessageAttributeAll,
			}),
		})
		if err != nil {
			logger.Errorf("failed to receive messages: %v, sleeping for %v", err, c.retry.wait)
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

		logger.Debugf("Consume: received %d messages", len(output.Messages))
		messageCountInc(ctx, c.queue.name, fetchedMessageState, false, len(output.Messages))

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
		observerMessageAge(c.queue.name, msg.Attributes)

		corID := getCorrelationID(msg.MessageAttributes)

		sp, ctxCh := trace.ConsumerSpan(ctx, trace.ComponentOpName(consumerComponent, c.queue.name),
			consumerComponent, corID, mapHeader(msg.MessageAttributes))

		ctxCh = correlation.ContextWithID(ctxCh, corID)
		logger := log.Sub(map[string]interface{}{correlation.ID: corID})
		ctxCh = log.WithContext(ctxCh, logger)

		btc.messages = append(btc.messages, message{
			ctx:   ctxCh,
			queue: c.queue,
			api:   c.api,
			msg:   msg,
			span:  sp,
		})
	}

	return btc
}

func (c *Component) report(ctx context.Context, sqsAPI sqsiface.SQSAPI, queueURL string) error {
	log.Debugf("retrieve stats for SQS %s", c.queue.name)
	rsp, err := sqsAPI.GetQueueAttributesWithContext(ctx, &sqs.GetQueueAttributesInput{
		AttributeNames: []*string{
			aws.String(sqsAttributeApproximateNumberOfMessages),
			aws.String(sqsAttributeApproximateNumberOfMessagesDelayed),
			aws.String(sqsAttributeApproximateNumberOfMessagesNotVisible),
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
	queueSize.WithLabelValues("available").Set(size)

	size, err = getAttributeFloat64(rsp.Attributes, sqsAttributeApproximateNumberOfMessagesDelayed)
	if err != nil {
		return err
	}
	queueSize.WithLabelValues("delayed").Set(size)

	size, err = getAttributeFloat64(rsp.Attributes, sqsAttributeApproximateNumberOfMessagesNotVisible)
	if err != nil {
		return err
	}
	queueSize.WithLabelValues("invisible").Set(size)
	return nil
}

func getAttributeFloat64(attr map[string]*string, key string) (float64, error) {
	valueString := attr[key]
	if valueString == nil {
		return 0.0, fmt.Errorf("value of %s does not exist", key)
	}
	value, err := strconv.ParseFloat(*valueString, 64)
	if err != nil {
		return 0.0, fmt.Errorf("could not convert %s to float64", *valueString)
	}
	return value, nil
}

func observerMessageAge(queue string, attributes map[string]*string) {
	attribute, ok := attributes[sqsAttributeSentTimestamp]
	if !ok || attribute == nil {
		return
	}
	timestamp, err := strconv.ParseInt(*attribute, 10, 64)
	if err != nil {
		return
	}
	messageAge.WithLabelValues(queue).Set(time.Now().UTC().Sub(time.Unix(timestamp, 0)).Seconds())
}

func messageCountInc(ctx context.Context, queue string, state messageState, hasError bool, count int) {
	hasErrorString := "false"
	if hasError {
		hasErrorString = "true"
	}

	messageCounter := trace.Counter{
		Counter: messageCounterVec.WithLabelValues(queue, string(state), hasErrorString),
	}
	messageCounter.Add(ctx, float64(count))
}

func getCorrelationID(ma map[string]*sqs.MessageAttributeValue) string {
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

func mapHeader(ma map[string]*sqs.MessageAttributeValue) map[string]string {
	mp := make(map[string]string)
	for key, value := range ma {
		if value.StringValue != nil {
			mp[key] = *value.StringValue
		}
	}
	return mp
}
