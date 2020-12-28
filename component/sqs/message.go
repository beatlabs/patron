package sqs

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
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
	Message() *sqs.Message
	// Span contains the tracing span of this message.
	Span() opentracing.Span
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
	api   sqsiface.SQSAPI
	msg   *sqs.Message
	span  opentracing.Span
}

func (m message) Context() context.Context {
	return m.ctx
}

func (m message) ID() string {
	return aws.StringValue(m.msg.MessageId)
}

func (m message) Body() []byte {
	return []byte(*m.msg.Body)
}

func (m message) Span() opentracing.Span {
	return m.span
}

func (m message) Message() *sqs.Message {
	return m.msg
}

func (m message) ACK() error {
	_, err := m.api.DeleteMessageWithContext(m.ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(m.queue.url),
		ReceiptHandle: m.msg.ReceiptHandle,
	})
	if err != nil {
		messageCountErrorInc(m.queue.name, ackMessageState, 1)
		trace.SpanError(m.span)
		return err
	}
	messageCountInc(m.queue.name, ackMessageState, 1)
	trace.SpanSuccess(m.span)
	return nil
}

func (m message) NACK() {
	messageCountInc(m.queue.name, nackMessageState, 1)
	trace.SpanSuccess(m.span)
}

type batch struct {
	ctx      context.Context
	queue    queue
	sqsAPI   sqsiface.SQSAPI
	messages []Message
}

func (b batch) ACK() ([]Message, error) {
	entries := make([]*sqs.DeleteMessageBatchRequestEntry, 0, len(b.messages))
	msgMap := make(map[string]Message, len(b.messages))

	for _, msg := range b.messages {
		entries = append(entries, &sqs.DeleteMessageBatchRequestEntry{
			Id:            aws.String(msg.ID()),
			ReceiptHandle: msg.Message().ReceiptHandle,
		})
		msgMap[msg.ID()] = msg
	}

	output, err := b.sqsAPI.DeleteMessageBatchWithContext(b.ctx, &sqs.DeleteMessageBatchInput{
		Entries:  entries,
		QueueUrl: aws.String(b.queue.url),
	})
	if err != nil {
		messageCountErrorInc(b.queue.name, ackMessageState, len(b.messages))
		for _, msg := range b.messages {
			trace.SpanError(msg.Span())
		}
		return nil, err
	}

	if len(output.Successful) > 0 {
		messageCountInc(b.queue.name, ackMessageState, len(output.Successful))

		for _, suc := range output.Successful {
			trace.SpanSuccess(msgMap[aws.StringValue(suc.Id)].Span())
		}
	}

	if len(output.Failed) > 0 {
		messageCountErrorInc(b.queue.name, ackMessageState, len(output.Failed))
		failed := make([]Message, 0, len(output.Failed))
		for _, fail := range output.Failed {
			msg := msgMap[aws.StringValue(fail.Id)]
			trace.SpanError(msg.Span())
			failed = append(failed, msg)
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
