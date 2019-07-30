package sqs

import (
	"context"

	"github.com/beatlabs/patron/trace"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/log"
	opentracing "github.com/opentracing/opentracing-go"
)

type message struct {
	sqs  *sqs.SQS
	ctx  context.Context
	msg  *sqs.Message
	span opentracing.Span
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
	// TODO: delete the message from SQS.
	// TODO: metrics
	trace.SpanSuccess(m.span)
	return nil
}

// Nack the message. SQS does not support Nack, the message will be available after the visibility timeout has passed.
func (m *message) Nack() error {
	// TODO: metrics
	log.Debugf("message %s not deleted, will be available", *m.msg.MessageId)
	trace.SpanError(m.span)
	return nil
}

// Factory for creating SQS consumers.
type Factory struct {
	queueURL        string
	maxMessages     int64
	pollWaitSeconds int64
	buffer          int
}

// Create a new SQS consumer.
func (f *Factory) Create() (async.Consumer, error) {
	//TODO: create sqs
	var sqs *sqs.SQS

	return &consumer{
		queueURL:        f.queueURL,
		maxMessages:     f.maxMessages,
		pollWaitSeconds: f.pollWaitSeconds,
		buffer:          f.buffer,
		sqs:             sqs,
	}, nil
}

type consumer struct {
	queueURL        string
	maxMessages     int64
	pollWaitSeconds int64
	buffer          int
	sqs             *sqs.SQS
	cnl             context.CancelFunc
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
					trace.KafkaConsumerComponent,
					mapHeader(msg.MessageAttributes),
				)

				chMsg <- &message{
					span: sp,
					msg:  msg,
					ctx:  chCtx,
					sqs:  c.sqs,
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
		mp[key] = value.string(a.Value)
	}
	return mp
}
