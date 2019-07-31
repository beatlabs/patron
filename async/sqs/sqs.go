package sqs

import (
	"context"

	"github.com/beatlabs/patron/errors"

	"github.com/beatlabs/patron/trace"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/log"
	opentracing "github.com/opentracing/opentracing-go"
)

type message struct {
	queueURL string
	sqs      *sqs.SQS
	ctx      context.Context
	msg      *sqs.Message
	span     opentracing.Span
}

// Context of the message.
func (m *message) Context() context.Context {
	return m.ctx
}

// Decode the message to the provided argument.
func (m *message) Decode(v interface{}) error {
	panic("implement me")
}

// Ack the message.
func (m *message) Ack() error {
	output, err := m.sqs.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(m.queueURL),
		ReceiptHandle: m.msg.ReceiptHandle,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to delete message %s from queue %s", *m.msg.MessageId)
	}
	log.Debugf("message %s deleted with output: %s", m.msg.MessageId, output.String())

	// TODO: metrics
	trace.SpanSuccess(m.span)
	return nil
}

// Nack the message. SQS does not support Nack, the message will be available after the visibility timeout has passed.
// We could investigate to support ChangeMessageVisibility which could be used to make the message visible again sooner
// than the visibility timeout.
func (m *message) Nack() error {

	// TODO: metrics
	trace.SpanError(m.span)
	log.Debugf("message %s not deleted, will be available after visibility timeout passes", *m.msg.MessageId)
	return nil
}

// Factory for creating SQS consumers.
type Factory struct {
	queueURL          string
	maxMessages       int64
	pollWaitSeconds   int64
	visibilityTimeout int64
	buffer            int
}

func NewFactory(queueURL string, oo ...OptionFunc) (*Factory, error) {
	if queueURL == "" {
		return nil, errors.New("queue url is empty")
	}

	f := &Factory{
		queueURL:          queueURL,
		maxMessages:       10, // maximum 10 messages per request
		pollWaitSeconds:   20, // poll wait time for long polling in seconds
		visibilityTimeout: 30, // the time in seconds after a message gets visible again after fetching it from the queue
		buffer:            0,  // this means that a message will be processed only if the previous has been processed
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
	//TODO: create sqs
	var sqs *sqs.SQS

	return &consumer{
		queueURL:          f.queueURL,
		maxMessages:       f.maxMessages,
		pollWaitSeconds:   f.pollWaitSeconds,
		buffer:            f.buffer,
		visibilityTimeout: f.visibilityTimeout,
		sqs:               sqs,
	}, nil
}

type consumer struct {
	queueURL          string
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
	go func() {
		for {
			if sqsCtx.Err() != nil {
				return
			}
			output, err := c.sqs.ReceiveMessageWithContext(sqsCtx, &sqs.ReceiveMessageInput{
				QueueUrl:            &c.queueURL,
				MaxNumberOfMessages: aws.Int64(c.maxMessages),
				WaitTimeSeconds:     aws.Int64(c.pollWaitSeconds),
				VisibilityTimeout:   aws.Int64(c.visibilityTimeout),
				// TODO: Use attributes to fetch the process time of a message...
			})
			if err != nil {
				chErr <- err
				continue
			}
			if sqsCtx.Err() != nil {
				return
			}
			for _, msg := range output.Messages {

				sp, chCtx := trace.ConsumerSpan(
					sqsCtx,
					trace.ComponentOpName(trace.SQSConsumerComponent, c.queueURL),
					trace.SQSConsumerComponent,
					mapHeader(msg.MessageAttributes),
				)

				chMsg <- &message{
					queueURL: c.queueURL,
					span:     sp,
					msg:      msg,
					ctx:      chCtx,
					sqs:      c.sqs,
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

func mapHeader(ma map[string]*sqs.MessageAttributeValue) map[string]string {
	mp := make(map[string]string)
	for key, value := range ma {
		mp[key] = *value.StringValue
	}
	return mp
}
