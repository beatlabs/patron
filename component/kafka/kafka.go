// Package kafka provides some shared interfaces for the Kafka components.
package kafka

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/trace"
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
	// Record will contain the raw Kafka record.
	Record() *kgo.Record
	// Span contains the tracing span of this message.
	Span() trace.Span
}

// NewMessage initializes a new message which is an implementation of the kafka Message interface.
func NewMessage(ctx context.Context, sp trace.Span, rec *kgo.Record) Message {
	return &message{
		ctx: ctx,
		sp:  sp,
		rec: rec,
	}
}

type message struct {
	ctx context.Context
	sp  trace.Span
	rec *kgo.Record
}

// Context will contain the context to be used for processing.
// Each context will have a logger setup which can be used to create a logger from context.
func (m *message) Context() context.Context {
	return m.ctx
}

// Record will contain the raw Kafka record.
func (m *message) Record() *kgo.Record {
	return m.rec
}

// Span contains the tracing span of this message.
func (m *message) Span() trace.Span {
	return m.sp
}

// Batch interface for multiple Kafka messages.
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

// DefaultConsumerConfig creates default franz-go options for a consumer with a client ID derived from hostname and consumer name.
func DefaultConsumerConfig(name string, readCommitted bool) ([]kgo.Opt, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, errors.New("failed to get hostname")
	}

	opts := []kgo.Opt{
		kgo.ClientID(fmt.Sprintf("%s-%s", host, name)),
	}

	if readCommitted {
		opts = append(opts, kgo.FetchIsolationLevel(kgo.ReadCommitted()))
	}

	return opts, nil
}
