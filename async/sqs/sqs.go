package sqs

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/log"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type message struct {
	msg *sqs.Message
}

// Context of the message.
func (m *message) Context() context.Context {
	panic("implement me")
}

// Decode the message to the provided argument.
func (m *message) Decode(v interface{}) error {
	panic("implement me")
}

// Ack the message.
func (m *message) Ack() error {
	// TODO: delete the message from SQS.
	// TODO: metrics
	return nil
}

// Nack the message. SQS does not support Nack, the message will be available after the visibility timeout has passed.
func (m *message) Nack() error {
	// TODO: use the correct message id from SQS
	// TODO: metrics
	log.Debugf("message %s not deleted, will be available", "1")
	return nil
}

// Factory for creating SQS consumers.
type Factory struct {
}

// Create a new SQS consumer.
func (f *Factory) Create() (async.Consumer, error) {
	panic("implement me")
}

type consumer struct {
	queueURL string
	maxMessages int64
	pollWaitSeconds int64
	buffer      int
  	sqs sqs.SQS
	cnl context.CancelFunc
}


// Consume messages from SQS and send them to the channel.
func (c *consumer) Consume(ctx context.Context) (<-chan async.Message, <-chan error, error) {
	chMsg := make(chan async.Message, c.buffer)
	chErr := make(chan error, c.buffer)
	sqsCtx,cnl:=context.WithCancel(ctx)
	c.cnl = cnl
	go func() {
		for {
			if sqsCtx.Err() != nil {
				return
			}
			output, err := c.sqs.ReceiveMessageWithContext(sqsCtx ,&sqs.ReceiveMessageInput{
				QueueUrl:            &c.queueURL,
				MaxNumberOfMessages: aws.Int64(c.maxMessages),
				WaitTimeSeconds:     aws.Int64(c.pollWaitSeconds),
			})
			if err != nil {
				chErr <-err
				continue
			}
			if sqsCtx.Err() != nil {
				return
			}
			for _, msg := range output.Messages {
				chMsg <- &message{msg:msg}
			}
		}
	}()
	return chMsg,chErr,nil
}

// Close the consumer.
func (c *consumer) Close() error {
	c.cnl()
	return nil
}
