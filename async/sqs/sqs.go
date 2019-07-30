package sqs

import (
	"context"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/log"
)

type message struct {
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
}

// Consume messages from SQS and send them to the channel.
func (c *consumer) Consume(context.Context) (<-chan async.Message, <-chan error, error) {
	panic("implement me")
}

// Close the consumer.
func (c *consumer) Close() error {
	panic("implement me")
}
