package sqs

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
)

var messageAge *prometheus.GaugeVec
var messageCounter *prometheus.CounterVec
var queueSize *prometheus.GaugeVec

func init() {
	messageAge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "component",
			Subsystem: "sqs_consumer",
			Name:      "message_age",
			Help:      "Message age based on the SentTimestamp SQS attribute",
		},
		[]string{"queue"},
	)
	prometheus.MustRegister(messageAge)
	messageCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "sqs_consumer",
			Name:      "message_counter",
			Help:      "Message counter",
		},
		[]string{"queue", "state", "hasError"},
	)
	prometheus.MustRegister(messageCounter)
	queueSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "component",
			Subsystem: "sqs_consumer",
			Name:      "queue_size",
			Help:      "Queue size reported by AWS",
		},
		[]string{"state"},
	)
	prometheus.MustRegister(queueSize)
}

type message struct {
	queue    string
	queueURL string
	sqs      sqsiface.SQSAPI
	ctx      context.Context
	msg      *sqs.Message
	span     opentracing.Span
	dec      encoding.DecodeRawFunc
}

// Context of the message.
func (m *message) Context() context.Context {
	return m.ctx
}

// Decode the message to the provided argument.
func (m *message) Decode(v interface{}) error {
	return m.dec([]byte(*m.msg.Body), v)
}

// Ack the message.
func (m *message) Ack() error {
	_, err := m.sqs.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(m.queueURL),
		ReceiptHandle: m.msg.ReceiptHandle,
	})
	if err != nil {
		messageCountErrorInc(m.queue, "ACK", 1)
		return nil
	}
	messageCountInc(m.queue, "ACK", 1)
	trace.SpanSuccess(m.span)
	return nil
}

// Nack the message. SQS does not support Nack, the message will be available after the visibility timeout has passed.
// We could investigate to support ChangeMessageVisibility which could be used to make the message visible again sooner
// than the visibility timeout.
func (m *message) Nack() error {
	messageCountInc(m.queue, "NACK", 1)
	trace.SpanError(m.span)
	return nil
}

// Factory for creating SQS consumers.
type Factory struct {
	queueName         string
	queueUrl          string
	maxMessages       int64
	pollWaitSeconds   int64
	visibilityTimeout int64
	buffer            int
	statsInterval     time.Duration
	queue             sqsiface.SQSAPI
}

// NewFactory creates a new consumer factory.
func NewFactory(queue sqsiface.SQSAPI, queueName string, oo ...OptionFunc) (*Factory, error) {
	if queue == nil {
		return nil, errors.New("queue is nil")
	}

	if queueName == "" {
		return nil, errors.New("queue name is empty")
	}

	url, err := queue.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return nil, err
	}

	f := &Factory{
		queueName:         queueName,
		queueUrl:          *url.QueueUrl,
		queue:             queue,
		maxMessages:       10,
		pollWaitSeconds:   20,
		visibilityTimeout: 30,
		buffer:            0,
		statsInterval:     10 * time.Second,
	}

	for _, o := range oo {
		err := o(f)
		if err != nil {
			return nil, err
		}
	}

	return f, nil
}

// Create a new SQS consumer.
func (f *Factory) Create() (async.Consumer, error) {
	return &consumer{
		queueName:         f.queueName,
		maxMessages:       f.maxMessages,
		pollWaitSeconds:   f.pollWaitSeconds,
		buffer:            f.buffer,
		visibilityTimeout: f.visibilityTimeout,
		statsInterval:     f.statsInterval,
		queue:             f.queue,
	}, nil
}

type consumer struct {
	queueName         string
	queueUrl          string
	queue             sqsiface.SQSAPI
	maxMessages       int64
	pollWaitSeconds   int64
	visibilityTimeout int64
	buffer            int
	statsInterval     time.Duration

	cnl context.CancelFunc
}

// Consume messages from SQS and send them to the channel.
func (c *consumer) Consume(ctx context.Context) (<-chan async.Message, <-chan error, error) {
	chMsg := make(chan async.Message, c.buffer)
	chErr := make(chan error, c.buffer)
	sqsCtx, cnl := context.WithCancel(ctx)
	c.cnl = cnl

	go func() {
		for {
			if sqsCtx.Err() != nil {
				return
			}
			log.Debugf("polling SQS queue %s for messages", c.queue)
			output, err := c.queue.ReceiveMessageWithContext(sqsCtx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(c.queueUrl),
				MaxNumberOfMessages: aws.Int64(c.maxMessages),
				WaitTimeSeconds:     aws.Int64(c.pollWaitSeconds),
				VisibilityTimeout:   aws.Int64(c.visibilityTimeout),
				AttributeNames: aws.StringSlice([]string{
					"SentTimestamp",
				}),
				MessageAttributeNames: aws.StringSlice([]string{
					"All",
				}),
			})
			if err != nil {
				chErr <- err
				continue
			}
			if sqsCtx.Err() != nil {
				return
			}

			messageCountInc(c.queueName, "FETCHED", len(output.Messages))

			for _, msg := range output.Messages {
				observerMessageAge(c.queueName, msg.Attributes)

				sp, chCtx := trace.ConsumerSpan(sqsCtx, trace.ComponentOpName(trace.SQSConsumerComponent, c.queueName),
					trace.SQSConsumerComponent, mapHeader(msg.MessageAttributes))

				ct, err := determineContentType(msg.MessageAttributes)
				if err != nil {
					messageCountErrorInc(c.queueName, "FETCHED", 1)
					trace.SpanError(sp)
					log.Errorf("failed to determine content type: %v", err)
					continue
				}

				dec, err := async.DetermineDecoder(ct)
				if err != nil {
					messageCountErrorInc(c.queueName, "FETCHED", 1)
					trace.SpanError(sp)
					log.Errorf("failed to determine decoder: %v", err)
					continue
				}

				chMsg <- &message{
					queue:    c.queueName,
					queueURL: c.queueUrl,
					span:     sp,
					msg:      msg,
					ctx:      log.WithContext(chCtx, log.Sub(map[string]interface{}{"messageID": *msg.MessageId})),
					sqs:      c.queue,
					dec:      dec,
				}
			}
		}
	}()
	go func() {
		tickerStats := time.NewTicker(c.statsInterval)
		defer tickerStats.Stop()
		for {
			select {
			case <-sqsCtx.Done():
				return
			case <-tickerStats.C:
				err := c.reportQueueStats(sqsCtx, c.queueUrl)
				if err != nil {
					log.Errorf("failed to report queue stats: %v", err)
				}
			}
		}
	}()
	return chMsg, chErr, nil
}

// Close the consumer.
func (c *consumer) Close() error {
	c.cnl()
	return nil
}

func (c *consumer) reportQueueStats(ctx context.Context, queueURL string) error {
	log.Debugf("retrieve stats for SQS %s", c.queue)
	rsp, err := c.queue.GetQueueAttributesWithContext(ctx, &sqs.GetQueueAttributesInput{
		AttributeNames: []*string{
			aws.String("ApproximateNumberOfMessages"),
			aws.String("ApproximateNumberOfMessagesDelayed"),
			aws.String("ApproximateNumberOfMessagesNotVisible")},
		QueueUrl: aws.String(queueURL),
	})
	if err != nil {
		return err
	}

	size, err := getAttributeFloat64(rsp.Attributes, "ApproximateNumberOfMessages")
	if err != nil {
		return err
	}
	queueSize.WithLabelValues("available").Set(size)

	size, err = getAttributeFloat64(rsp.Attributes, "ApproximateNumberOfMessagesDelayed")
	if err != nil {
		return err
	}
	queueSize.WithLabelValues("delayed").Set(size)

	size, err = getAttributeFloat64(rsp.Attributes, "ApproximateNumberOfMessagesNotVisible")
	if err != nil {
		return err
	}
	queueSize.WithLabelValues("invisible").Set(size)
	return nil
}

func getAttributeFloat64(attr map[string]*string, key string) (float64, error) {
	valueString := attr[key]
	if valueString == nil {
		return 0.0, errors.Errorf("value of %s does not exist", key)
	}
	value, err := strconv.ParseFloat(*valueString, 64)
	if err != nil {
		return 0.0, errors.Errorf("could not convert %s to float64", *valueString)
	}
	return value, nil
}

func determineContentType(ma map[string]*sqs.MessageAttributeValue) (string, error) {
	for key, value := range ma {
		if key == encoding.ContentTypeHeader {
			if value.StringValue != nil {
				return *value.StringValue, nil
			}
			return "", errors.New("content type header is nil")
		}
	}
	return json.Type, nil
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

func observerMessageAge(queue string, attributes map[string]*string) {
	attribute, ok := attributes["SentTimestamp"]
	if !ok || attribute == nil {
		return
	}
	timestamp, err := strconv.ParseInt(*attribute, 10, 64)
	if err != nil {
		return
	}
	messageAge.WithLabelValues(queue).Set(time.Now().UTC().Sub(time.Unix(timestamp, 0)).Seconds())
}

func messageCountInc(queue, state string, count int) {
	messageCounter.WithLabelValues(queue, state, "false").Add(float64(count))
}

func messageCountErrorInc(queue, state string, count int) {
	messageCounter.WithLabelValues(queue, state, "true").Add(float64(count))
}
