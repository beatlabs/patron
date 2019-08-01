package sqs

import (
	"context"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
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
			Name:      "message_age",
			Help:      "Message age based on the SentTimestamp SQS attribute",
		},
		[]string{"queue", "state", "hasError"},
	)
	prometheus.MustRegister(messageCounter)
}

type message struct {
	queue    string
	queueURL *string
	sqs      *sqs.SQS
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
		QueueUrl:      m.queueURL,
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

type Config struct {
	region string
	id     string
	secret string
	token  string
}

func NewConfig(region, id, secret, token string) (*Config, error) {
	if region == "" {
		return nil, errors.New("AWS region not provided")
	}
	if id == "" {
		return nil, errors.New("AWS id not provided")
	}
	if secret == "" {
		return nil, errors.New("AWS secret not provided")
	}
	return &Config{
		region: region,
		id:     id,
		secret: secret,
		token:  token,
	}, nil
}

// Factory for creating SQS consumers.
type Factory struct {
	queue             string
	maxMessages       int64
	pollWaitSeconds   int64
	visibilityTimeout int64
	buffer            int
	ses               *session.Session
}

func NewFactory(cfg Config, queue string, oo ...OptionFunc) (*Factory, error) {
	if queue == "" {
		return nil, errors.New("queue name is empty")
	}
	ses, err := session.NewSession(&aws.Config{
		Region:      aws.String(cfg.region),
		Credentials: credentials.NewStaticCredentials(cfg.id, cfg.secret, cfg.token),
	})
	if err != nil {
		return nil, err
	}

	f := &Factory{
		queue:             queue,
		maxMessages:       10,
		pollWaitSeconds:   20,
		visibilityTimeout: 30,
		buffer:            0,
		ses:               ses,
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
		queue:             f.queue,
		maxMessages:       f.maxMessages,
		pollWaitSeconds:   f.pollWaitSeconds,
		buffer:            f.buffer,
		visibilityTimeout: f.visibilityTimeout,
		sqs:               sqs.New(f.ses),
	}, nil
}

type consumer struct {
	queue             string
	maxMessages       int64
	pollWaitSeconds   int64
	visibilityTimeout int64
	buffer            int
	sqs               *sqs.SQS
	cnl               context.CancelFunc
}

// Consume messages from SQS and send them to the channel.
func (c *consumer) Consume(ctx context.Context) (<-chan async.Message, <-chan error, error) {
	chMsg := make(chan async.Message, c.buffer)
	chErr := make(chan error, c.buffer)
	sqsCtx, cnl := context.WithCancel(ctx)
	c.cnl = cnl

	queueURL, err := c.sqs.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(c.queue),
	})
	if err != nil {
		return nil, nil, err
	}

	go func() {
		for {
			if sqsCtx.Err() != nil {
				return
			}
			output, err := c.sqs.ReceiveMessageWithContext(sqsCtx, &sqs.ReceiveMessageInput{
				QueueUrl:            queueURL.QueueUrl,
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

			messageCountInc(c.queue, "FETCHED", len(output.Messages))

			for _, msg := range output.Messages {
				observerMessageAge(c.queue, msg.Attributes)

				sp, chCtx := trace.ConsumerSpan(sqsCtx, trace.ComponentOpName(trace.SQSConsumerComponent, c.queue),
					trace.SQSConsumerComponent, mapHeader(msg.MessageAttributes))

				ct, err := determineContentType(msg.MessageAttributes)
				if err != nil {
					messageCountErrorInc(c.queue, "FETCHED", 1)
					trace.SpanError(sp)
					log.Errorf("failed to determine content type: %v", err)
					continue
				}

				dec, err := async.DetermineDecoder(ct)
				if err != nil {
					messageCountErrorInc(c.queue, "FETCHED", 1)
					trace.SpanError(sp)
					log.Errorf("failed to determine decoder: %v", err)
					continue
				}

				chMsg <- &message{
					queue:    c.queue,
					queueURL: queueURL.QueueUrl,
					span:     sp,
					msg:      msg,
					ctx:      log.WithContext(chCtx, log.Sub(map[string]interface{}{"messageID": *msg.MessageId})),
					sqs:      c.sqs,
					dec:      dec,
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
