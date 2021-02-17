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
	// Commit marks and then commits the batch offset in a synchronous blocking operation.
	Commit()
	// MarkBatchOffset marks the batch offset in an asynchronous operation.
	// AutoCommit needs to be turned on in order to actually commits offset.
	// This can be done by setting Consumer.Offsets.AutoCommit.Enable = true in the sarama configuration.
	// It is true by default.
	MarkBatchOffset()
}

// commitFunc definition of the batch commit function.
type commitFunc func()

// markBatchOffsetFunc definition of the batch mark offset function.
type markBatchOffsetFunc func(messages []Message)

type batch struct {
	messages            []Message
	commitFunc          commitFunc
	markBatchOffsetFunc markBatchOffsetFunc
}

// Messages of the batch.
func (b batch) Messages() []Message {
	return b.messages
}

// Commit marks and then commits the batch offset in a synchronous blocking operation.
func (b batch) Commit() {
	b.MarkBatchOffset()
	b.commitFunc()
}

// MarkBatchOffset the batch offset in an asynchronous operation.
func (b batch) MarkBatchOffset() {
	b.markBatchOffsetFunc(b.messages)
}

// defaultSaramaConfig function creates a sarama config object with the default configuration set up.
func defaultSaramaConfig(name string) (*sarama.Config, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, errors.New("failed to get hostname")
	}

	config := sarama.NewConfig()
	config.ClientID = fmt.Sprintf("%s-%s", host, name)
	config.Consumer.Return.Errors = true
	config.Version = sarama.V0_11_0_0

	return config, nil
}
