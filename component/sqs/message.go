package sqs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"go.opentelemetry.io/otel/trace"
)

// Message interface for AWS SQS message.
type Message interface {
	// Context will contain the context to be used for processing.
	// Each context will have a logger setup which can be used to create a logger from context.
	Context() context.Context
	// ID of the message.
	ID() string
	// Body of the message.
	Body() []byte
	// Message will contain the raw SQS message.
	Message() types.Message
	// Span contains the tracing span of this message.
	Span() trace.Span
	// ACK deletes the message from the queue and completes the tracing span.
	ACK() error
	// NACK leaves the message in the queue and completes the tracing span.
	NACK()
}

// Batch interface for multiple AWS SQS messages.
type Batch interface {
	// Messages of the batch.
	Messages() []Message
	// ACK deletes all messages from SQS with a single call and completes the all the message tracing spans.
	// In case the action will not manage to ACK all the messages, a slice of the failed messages will be returned.
	ACK() ([]Message, error)
	// NACK leaves all messages in the queue and completes the all the message tracing spans.
	NACK()
}

type queue struct {
	name string
	url  string
}

type message struct {
	ctx   context.Context
	queue queue
	api   API
	msg   types.Message
	span  trace.Span
}

func (m message) Context() context.Context {
	return m.ctx
}

func (m message) ID() string {
	return aws.ToString(m.msg.MessageId)
}

func (m message) Body() []byte {
	return []byte(*m.msg.Body)
}

func (m message) Span() trace.Span {
	return m.span
}

func (m message) Message() types.Message {
	return m.msg
}

func (m message) ACK() error {
	defer m.span.End()

	_, err := m.api.DeleteMessage(m.ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(m.queue.url),
		ReceiptHandle: m.msg.ReceiptHandle,
	})
	if err != nil {
		observeMessageCount(m.ctx, m.queue.name, ackMessageState, err, 1)
		patrontrace.SetSpanError(m.span, "failed to ACK message", err)
		return err
	}
	observeMessageCount(m.ctx, m.queue.name, ackMessageState, nil, 1)
	patrontrace.SetSpanSuccess(m.span)
	return nil
}

func (m message) NACK() {
	defer m.span.End()
	observeMessageCount(m.ctx, m.queue.name, nackMessageState, nil, 1)
	patrontrace.SetSpanSuccess(m.span)
}

type batch struct {
	ctx      context.Context
	queue    queue
	sqsAPI   API
	messages []Message
}

func (b batch) ACK() ([]Message, error) {
	entries := make([]types.DeleteMessageBatchRequestEntry, 0, len(b.messages))
	msgMap := make(map[string]Message, len(b.messages))

	for _, msg := range b.messages {
		entries = append(entries, types.DeleteMessageBatchRequestEntry{
			Id:            aws.String(msg.ID()),
			ReceiptHandle: msg.Message().ReceiptHandle,
		})
		msgMap[msg.ID()] = msg
	}

	output, err := b.sqsAPI.DeleteMessageBatch(b.ctx, &sqs.DeleteMessageBatchInput{
		Entries:  entries,
		QueueUrl: aws.String(b.queue.url),
	})
	if err != nil {
		observeMessageCount(b.ctx, b.queue.name, ackMessageState, err, len(b.messages))
		for _, msg := range b.messages {
			patrontrace.SetSpanError(msg.Span(), "failed to ACK message", err)
			msg.Span().End()
		}
		return nil, err
	}

	if len(output.Successful) > 0 {
		observeMessageCount(b.ctx, b.queue.name, ackMessageState, nil, len(output.Successful))

		for _, suc := range output.Successful {
			sp := msgMap[aws.ToString(suc.Id)].Span()
			patrontrace.SetSpanSuccess(sp)
			sp.End()
		}
	}

	if len(output.Failed) > 0 {
		observeMessageCount(b.ctx, b.queue.name, ackMessageState, nil, len(output.Failed))
		failed := make([]Message, 0, len(output.Failed))
		for _, fail := range output.Failed {
			msg := msgMap[aws.ToString(fail.Id)]
			failureErr := fmt.Errorf("failure code: %s message: %s", *fail.Code, *fail.Message)
			patrontrace.SetSpanError(msg.Span(), "failed to ACK message", failureErr)
			failed = append(failed, msg)
			msg.Span().End()
		}
		return failed, nil
	}

	return nil, nil
}

func (b batch) NACK() {
	for _, msg := range b.messages {
		msg.NACK()
	}
}

func (b batch) Messages() []Message {
	return b.messages
}
