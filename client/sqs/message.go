package sqs

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const attributeDataTypeString string = "String"

// MessageBuilder helps building messages to be sent to SQS.
type MessageBuilder struct {
	err   error
	input *sqs.SendMessageInput
}

// NewMessageBuilder creates a new MessageBuilder that helps creating messages.
//
// Deprecated: The SQS client package is superseded by the `github.com/beatlabs/client/sqs/v2` package.
// Please refer to the documents and the examples for the usage.
//
// This package is frozen and no new functionality will be added.
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		input: &sqs.SendMessageInput{
			MessageAttributes: map[string]*sqs.MessageAttributeValue{},
		},
	}
}

// Message is a struct embedding information about messages that will
// be later published to SQS thanks to the SQS publisher.
type Message struct {
	input *sqs.SendMessageInput
}

// Build tries to build a message given its specified data and returns an error if any goes wrong.
func (b *MessageBuilder) Build() (*Message, error) {
	if b.err != nil {
		return nil, b.err
	}

	if b.input.MessageBody == nil {
		return nil, errors.New("missing required field: message body")
	}

	if b.input.QueueUrl == nil {
		return nil, errors.New("missing required field: message queue URL")
	}

	// Messages with either a group ID or deduplication ID can't have a delay.
	// These two attributes are only used for FIFO queues, which don't allow for individual message delays.
	if (b.input.MessageGroupId != nil || b.input.MessageDeduplicationId != nil) && b.input.DelaySeconds != nil {
		return nil, errors.New("could not set a delay with either a group ID or a deduplication ID")
	}

	for name, attributeValue := range b.input.MessageAttributes {
		if err := attributeValue.Validate(); err != nil {
			return nil, fmt.Errorf("invalid attribute %s: %w", name, err)
		}
	}

	return &Message{input: b.input}, nil
}

// Body sets the body of the message.
func (b *MessageBuilder) Body(body string) *MessageBuilder {
	b.input.SetMessageBody(body)
	return b
}

// QueueURL sets the queue URL.
func (b *MessageBuilder) QueueURL(url string) *MessageBuilder {
	b.input.SetQueueUrl(url)
	return b
}

// WithDeduplicationID sets the deduplication ID.
func (b *MessageBuilder) WithDeduplicationID(id string) *MessageBuilder {
	b.input.SetMessageDeduplicationId(id)
	return b
}

// WithGroupID sets the group ID.
func (b *MessageBuilder) WithGroupID(id string) *MessageBuilder {
	b.input.SetMessageGroupId(id)
	return b
}

// WithDelaySeconds sets the delay of the message, in seconds.
func (b *MessageBuilder) WithDelaySeconds(seconds int64) *MessageBuilder {
	b.input.SetDelaySeconds(seconds)
	return b
}

// injectHeaders injects the SQS headers carrier's headers into the message's attributes.
func (m *Message) injectHeaders(carrier sqsHeadersCarrier) error {
	for k, v := range carrier {
		val, ok := v.(string)
		if !ok {
			return errors.New("failed to type assert string")
		}
		m.setMessageAttribute(k, val)
	}
	return nil
}

func (m *Message) setMessageAttribute(key, value string) {
	m.input.MessageAttributes[key] = &sqs.MessageAttributeValue{
		DataType:    aws.String(attributeDataTypeString),
		StringValue: aws.String(value),
	}
}
