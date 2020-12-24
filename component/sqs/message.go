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
	Context() context.Context
	Message() *sqs.Message
	Span() opentracing.Span
	ACK() error
	NACK()
}

// Batch interface for multiple AWS SQS messages.
type Batch interface {
	Messages() []Message
	ACK() error
	NACK()
}

type message struct {
	ctx       context.Context
	queueName string
	queueURL  string
	queue     sqsiface.SQSAPI
	msg       *sqs.Message
	span      opentracing.Span
}

func (m message) Context() context.Context {
	return m.ctx
}

func (m message) Span() opentracing.Span {
	return m.span
}

func (m message) Message() *sqs.Message {
	return m.msg
}

func (m message) ACK() error {
	_, err := m.queue.DeleteMessageWithContext(m.ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(m.queueURL),
		ReceiptHandle: m.msg.ReceiptHandle,
	})
	if err != nil {
		messageCountErrorInc(m.queueName, ackMessageState, 1)
		trace.SpanError(m.span)
		return err
	}
	messageCountInc(m.queueName, ackMessageState, 1)
	trace.SpanSuccess(m.span)
	return nil
}

func (m message) NACK() {
	messageCountInc(m.queueName, nackMessageState, 1)
	trace.SpanSuccess(m.span)
}

type batch struct {
	ctx       context.Context
	queueName string
	queueURL  string
	sqsAPI    sqsiface.SQSAPI
	messages  []Message
}

func (b batch) ACK() error {
	entries := make([]*sqs.DeleteMessageBatchRequestEntry, 0, len(b.messages))
	msgMap := make(map[string]Message, len(b.messages))

	for _, msg := range b.messages {
		entries = append(entries, &sqs.DeleteMessageBatchRequestEntry{
			Id:            msg.Message().MessageId,
			ReceiptHandle: msg.Message().ReceiptHandle,
		})
		msgMap[aws.StringValue(msg.Message().MessageId)] = msg
	}

	output, err := b.sqsAPI.DeleteMessageBatchWithContext(b.ctx, &sqs.DeleteMessageBatchInput{
		Entries:  entries,
		QueueUrl: aws.String(b.queueURL),
	})
	if err != nil {
		messageCountErrorInc(b.queueName, ackMessageState, len(b.messages))
		for _, msg := range b.messages {
			trace.SpanError(msg.Span())
		}
		return err
	}

	if len(output.Successful) > 0 {
		messageCountInc(b.queueName, ackMessageState, len(output.Successful))

		for _, suc := range output.Successful {
			trace.SpanSuccess(msgMap[aws.StringValue(suc.Id)].Span())
		}
	}

	if len(output.Failed) > 0 {
		messageCountErrorInc(b.queueName, ackMessageState, len(output.Failed))

		for _, fail := range output.Failed {
			trace.SpanError(msgMap[aws.StringValue(fail.Id)].Span())
		}
	}

	return nil
}

func (b batch) NACK() {
	for _, msg := range b.messages {
		msg.NACK()
	}
}

func (b batch) Messages() []Message {
	return b.messages
}
