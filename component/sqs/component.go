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
	messageAge     *prometheus.GaugeVec
	messageCounter *prometheus.CounterVec
	queueSize      *prometheus.GaugeVec
)

func init() {
	messageAge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "component",
			Subsystem: "sqs",
			Name:      "message_age",
			Help:      "Message age based on the SentTimestamp SQS attribute",
		},
		[]string{"sqsAPI"},
	)
	prometheus.MustRegister(messageAge)
	messageCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "sqs",
			Name:      "message_counter",
			Help:      "Message counter",
		},
		[]string{"sqsAPI", "state", "hasError"},
	)
	prometheus.MustRegister(messageCounter)
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

// Component implementation of a async component.
type Component struct {
	name              string
	queueName         string
	queueURL          string
	sqsAPI            sqsiface.SQSAPI
	maxMessages       *int64
	pollWaitSeconds   *int64
	visibilityTimeout *int64
	statsInterval     time.Duration
	proc              ProcessorFunc
	retries           uint
	retryWait         time.Duration
}

// New creates a new component with support for functional configuration.
func New(name, queueName, queueURL string, sqsAPI sqsiface.SQSAPI, proc ProcessorFunc, oo ...OptionFunc) (*Component, error) {
	if name == "" {
		return nil, errors.New("component name is empty")
	}

	if queueName == "" {
		return nil, errors.New("queue name is empty")
	}

	if queueURL == "" {
		return nil, errors.New("queue URL is empty")
	}

	if sqsAPI == nil {
		return nil, errors.New("SQS API is nil")
	}

	if proc == nil {
		return nil, errors.New("process function is nil")
	}

	cmp := &Component{
		name:              name,
		queueName:         queueName,
		queueURL:          queueURL,
		sqsAPI:            sqsAPI,
		maxMessages:       aws.Int64(3),
		pollWaitSeconds:   nil,
		visibilityTimeout: nil,
		statsInterval:     10 * time.Second,
		proc:              proc,
		retries:           10,
		retryWait:         time.Second,
	}

	var err error
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

	tickerStats := time.NewTicker(c.statsInterval)
	defer tickerStats.Stop()
	for {
		select {
		case err := <-chErr:
			return err
		case <-ctx.Done():
			return nil
		case <-tickerStats.C:
			err := c.report(ctx, c.sqsAPI, c.queueURL)
			if err != nil {
				log.FromContext(ctx).Errorf("failed to report sqsAPI stats: %v", err)
			}
		}
	}
}

func (c *Component) consume(ctx context.Context, chErr chan error) {
	logger := log.FromContext(ctx)

	retries := c.retries

	for {
		if ctx.Err() != nil {
			return
		}
		logger.Debugf("consume: polling SQS sqsAPI %s for %d messages", c.queueName, *c.maxMessages)
		output, err := c.sqsAPI.ReceiveMessageWithContext(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &c.queueURL,
			MaxNumberOfMessages: c.maxMessages,
			WaitTimeSeconds:     c.pollWaitSeconds,
			VisibilityTimeout:   c.visibilityTimeout,
			AttributeNames: aws.StringSlice([]string{
				sqsAttributeSentTimestamp,
			}),
			MessageAttributeNames: aws.StringSlice([]string{
				sqsMessageAttributeAll,
			}),
		})
		if err != nil {
			logger.Errorf("failed to receive messages: %v, sleeping for %v", err, c.retryWait)
			time.Sleep(c.retryWait)
			retries--
			if retries > 0 {
				continue
			}
			chErr <- err
			return
		}
		retries = c.retries

		if ctx.Err() != nil {
			return
		}

		logger.Debugf("Consume: received %d messages", len(output.Messages))
		messageCountInc(c.queueName, fetchedMessageState, len(output.Messages))

		if len(output.Messages) == 0 {
			continue
		}

		btc := batch{
			ctx:       ctx,
			queueName: c.queueName,
			queueURL:  c.queueURL,
			sqsAPI:    c.sqsAPI,
			messages:  make([]Message, 0, len(output.Messages)),
		}

		for _, msg := range output.Messages {
			observerMessageAge(c.queueName, msg.Attributes)

			corID := getCorrelationID(msg.MessageAttributes)

			sp, ctxCh := trace.ConsumerSpan(ctx, trace.ComponentOpName(consumerComponent, c.queueName),
				consumerComponent, corID, mapHeader(msg.MessageAttributes))

			ctxCh = correlation.ContextWithID(ctxCh, corID)
			logger := log.Sub(map[string]interface{}{correlation.ID: corID})
			ctxCh = log.WithContext(ctxCh, logger)

			btc.messages = append(btc.messages, message{
				ctx:       ctxCh,
				queueName: c.queueName,
				queueURL:  c.queueURL,
				queue:     c.sqsAPI,
				msg:       msg,
				span:      sp,
			})
		}

		c.proc(ctx, btc)
	}
}

func (c *Component) report(ctx context.Context, sqsAPI sqsiface.SQSAPI, queueURL string) error {
	log.Debugf("retrieve stats for SQS %s", c.queueName)
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

func messageCountInc(queue string, state messageState, count int) {
	messageCounter.WithLabelValues(queue, string(state), "false").Add(float64(count))
}

func messageCountErrorInc(queue string, state messageState, count int) {
	messageCounter.WithLabelValues(queue, string(state), "true").Add(float64(count))
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
