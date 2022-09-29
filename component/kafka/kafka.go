// Package kafka provides some shared interfaces for the Kafka components.
package kafka

import (
	"context"
	"errors"
	"fmt"
	"os"

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

// NewMessage initializes a new message which is an implementation of the kafka Message interface.
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

// NewBatch initializes a new batch of messages returning an instance of the implementation of the kafka Batch interface.
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

// DefaultConsumerSaramaConfig function creates a Sarama configuration with a client ID derived from host name and consumer name.
func DefaultConsumerSaramaConfig(name string, readCommitted bool) (*sarama.Config, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, errors.New("failed to get hostname")
	}

	config := sarama.NewConfig()
	config.ClientID = fmt.Sprintf("%s-%s", host, name)
	config.Consumer.Return.Errors = true
	if readCommitted {
		// from Kafka documentation:
		// Transactions were introduced in Kafka 0.11.0 wherein applications can write to multiple topics and partitions atomically. In order for this to work, consumers reading from these partitions should be configured to only read committed data. This can be achieved by setting the isolation.level=read_committed in the consumer's configuration.
		// In read_committed mode, the consumer will read only those transactional messages which have been successfully committed. It will continue to read non-transactional messages as before. There is no client-side buffering in read_committed mode. Instead, the end offset of a partition for a read_committed consumer would be the offset of the first message in the partition belonging to an open transaction. This offset is known as the 'Last Stable Offset'(LSO).
		config.Consumer.IsolationLevel = sarama.ReadCommitted
	}

	return config, nil
}
