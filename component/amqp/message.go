package amqp

import (
	"context"

	patronerrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/streadway/amqp"
)

// Message interface for an AMQP Delivery.
type Message interface {
	// Context will contain the context to be used for processing.
	// Each context will have a logger setup which can be used to create a logger from context.
	Context() context.Context
	// ID of the message.
	ID() string
	// Body of the message.
	Body() []byte
	// Message will contain the raw AMQP delivery.
	Message() amqp.Delivery
	// Span contains the tracing span of this message.
	Span() opentracing.Span
	// ACK deletes the message from the queue and completes the tracing span.
	ACK() error
	// NACK leaves the message in the queue and completes the tracing span.
	NACK() error
}

// Batch interface for multiple AWS SQS messages.
type Batch interface {
	// Messages of the batch.
	Messages() []Message
	// ACK deletes all messages and completes the message tracing spans.
	// In case the action will not manage to ACK all the messages, a slice of the failed messages will be returned.
	ACK() ([]Message, error)
	// NACK leaves all messages in the queue and completes all message tracing spans.
	// In case the action will not manage to NACK all the messages, a slice of the failed messages will be returned.
	NACK() ([]Message, error)
}

type message struct {
	ctx     context.Context
	span    opentracing.Span
	msg     amqp.Delivery
	queue   string
	requeue bool
}

func (m message) Context() context.Context {
	return m.ctx
}

func (m message) ID() string {
	return m.msg.MessageId
}

func (m message) Body() []byte {
	return m.msg.Body
}

func (m message) Span() opentracing.Span {
	return m.span
}

func (m message) Message() amqp.Delivery {
	return m.msg
}

func (m message) ACK() error {
	err := m.msg.Ack(false)
	trace.SpanComplete(m.span, err)
	messageCountInc(m.ctx, m.queue, ackMessageState, err)
	return err
}

func (m message) NACK() error {
	err := m.msg.Nack(false, m.requeue)
	messageCountInc(m.ctx, m.queue, nackMessageState, err)
	trace.SpanComplete(m.span, err)
	return err
}

type batch struct {
	messages []Message
}

func (b *batch) ACK() ([]Message, error) {
	var errors []error
	var failed []Message

	for _, msg := range b.messages {
		err := msg.ACK()
		if err != nil {
			errors = append(errors, err)
			failed = append(failed, msg)
		}
	}

	return failed, patronerrors.Aggregate(errors...)
}

func (b *batch) NACK() ([]Message, error) {
	var errors []error
	var failed []Message

	for _, msg := range b.messages {
		err := msg.NACK()
		if err != nil {
			errors = append(errors, err)
			failed = append(failed, msg)
		}
	}

	return failed, patronerrors.Aggregate(errors...)
}

func (b *batch) Messages() []Message {
	return b.messages
}

func (b *batch) append(msg *message) {
	b.messages = append(b.messages, msg)
}

func (b *batch) reset() {
	b.messages = b.messages[:0]
}
