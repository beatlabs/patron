package sqs

import (
	"fmt"
	"strconv"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type attributeDataType string

const (
	attributeDataTypeString      attributeDataType = "String"
	attributeDataTypeNumber      attributeDataType = "Number"
	attributeDataTypeBinary      attributeDataType = "Binary"
	attributeDataTypeCustom      attributeDataType = "Custom" // TODO: implement
)

// MessageBuilder helps building messages to be sent to SQS.
type MessageBuilder struct {
	err   error
	input *sqs.SendMessageInput
}

// NewMessageBuilder creates a new MessageBuilder that helps creating messages.
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

// WithGroupID sets the group ID.
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

// WithStringAttribute attaches a string attribute to the message.
func (b *MessageBuilder) WithStringAttribute(name, value string) *MessageBuilder {
	v := b.addAttributeValue(name, attributeDataTypeString)
	v.SetStringValue(value)
	return b
}

// WithNumberAttribute attaches a number attribute to the message.
func (b *MessageBuilder) WithNumberAttribute(name string, value int64) *MessageBuilder {
	v := b.addAttributeValue(name, attributeDataTypeNumber)
	v.SetStringValue(strconv.FormatInt(value, 10)) // We need to use StringValue for Number (see SDK doc)
	return b
}

// WithBinaryAttribute attaches a binary attribute to the message.
func (b *MessageBuilder) WithBinaryAttribute(name string, value []byte) *MessageBuilder {
	v := b.addAttributeValue(name, attributeDataTypeBinary)
	v.SetBinaryValue(value)
	return b
}

// WithCustomAttribute attaches a custom attribute to the message.
func (b *MessageBuilder) WithCustomAttribute(name string, value []byte) *MessageBuilder {
	// TODO: implement this
	return b
}

func (b *MessageBuilder) addAttributeValue(name string, dataType attributeDataType) *sqs.MessageAttributeValue {
	attributeValue := &sqs.MessageAttributeValue{}
	attributeValue.SetDataType(string(dataType))
	b.input.MessageAttributes[name] = attributeValue
	return attributeValue
}

// injectHeaders injects the SQS headers carrier's headers into the message's attributes.
func (m *Message) injectHeaders(carrier sqsHeadersCarrier) {
	for k, v := range carrier {
		m.setMessageAttribute(k, v.(string))
	}
}

func (m *Message) setMessageAttribute(key, value string) {
	m.input.MessageAttributes[key] = &sqs.MessageAttributeValue{
		DataType:    aws.String(string(attributeDataTypeString)),
		StringValue: aws.String(value),
	}
}
