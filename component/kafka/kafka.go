// Package kafka provides some shared interfaces for the Kafka components.
package kafka

import (
	"context"

	"github.com/Shopify/sarama"
	"github.com/opentracing/opentracing-go"
)

// FailStrategy type definition.
type FailStrategy int

const (
	// ExitStrategy does not commit failed message offsets and exits the application.
	ExitStrategy FailStrategy = iota
	// SkipStrategy commits the offset of messages that failed processing, and continues processing.
	SkipStrategy
)

// BatchProcessorFunc definition of a batch async processor function.
type BatchProcessorFunc func(Batch) error

// Message interface for wrapping messages that are handled by the kafka component.
type Message interface {
	// Context will contain the context to be used for processing.
	// Each context will have a logger setup which can be used to create a logger from context.
	Context() context.Context
	// Message will contain the raw Kafka message.
	Message() *sarama.ConsumerMessage
	// Span contains the tracing span of this message.
	Span() opentracing.Span
}

// NewMessage initializes a new message which is an implementation of the kafka Message interface
func NewMessage(ctx context.Context, sp opentracing.Span, msg *sarama.ConsumerMessage) Message {
	return &message{
		ctx: ctx,
		sp:  sp,
		msg: msg,
	}
}

type message struct {
	ctx context.Context
	sp  opentracing.Span
	msg *sarama.ConsumerMessage
}

// Context will contain the context to be used for processing.
// Each context will have a logger setup which can be used to create a logger from context.
func (m *message) Context() context.Context {
	return m.ctx
}

// Message will contain the raw Kafka message.
func (m *message) Message() *sarama.ConsumerMessage {
	return m.msg
}

// Span contains the tracing span of this message.
func (m *message) Span() opentracing.Span {
	return m.sp
}

// Batch interface for multiple AWS SQS messages.
type Batch interface {
	// Messages of the batch.
	Messages() []Message
}

// NewBatch initializes a new batch of messages returning an instance of the implementation of the kafka Batch interface
func NewBatch(messages []Message) Batch {
	return &batch{
		messages: messages,
	}
}

type batch struct {
	messages []Message
}

// Messages of the batch.
func (b batch) Messages() []Message {
	return b.messages
}
