package sns

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
)

type attributeDataType string

const (
	attributeDataTypeString      attributeDataType = "String"
	attributeDataTypeStringArray attributeDataType = "StringArray"
	attributeDataTypeNumber      attributeDataType = "Number"
	attributeDataTypeBinary      attributeDataType = "Binary"

	tracingTargetTopicArn    = "topic-arn"
	tracingTargetTargetArn   = "target-arn"
	tracingTargetPhoneNumber = "phone-number"
	tracingTargetUnknown     = "unknown"
)

// MessageBuilder helps to build messages to be sent to SNS.
type MessageBuilder struct {
	err   error
	input *sns.PublishInput
}

// NewMessageBuilder creates a new MessageBuilder that helps to create messages.
//
// Deprecated: The SNS client package is superseded by the `github.com/beatlabs/client/sns/v2` package.
// Please refer to the documents and the examples for the usage.
//
// This package is frozen and no new functionality will be added.
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		input: &sns.PublishInput{
			MessageAttributes: map[string]*sns.MessageAttributeValue{},
		},
	}
}

// Message is a struct embedding information about messages that will
// be later published to SNS thanks to the SNS publisher.
type Message struct {
	input *sns.PublishInput
}

// tracingTarget returns a string used for tracing operations. As a message can only define one TopicArn,
// TargetArn or PhoneNumber, we set a different tracing topic for each case.
func (m Message) tracingTarget() string {
	if m.input.TopicArn != nil {
		return fmt.Sprintf("%s:%s", tracingTargetTopicArn, *m.input.TopicArn)
	}

	if m.input.TargetArn != nil {
		return fmt.Sprintf("%s:%s", tracingTargetTargetArn, *m.input.TargetArn)
	}

	if m.input.PhoneNumber != nil {
		// We don't append the phone number so that we don't end with 1 target per phone number.
		return tracingTargetPhoneNumber
	}

	return tracingTargetUnknown
}

// Message attaches a message to the message struct.
func (b *MessageBuilder) Message(msg string) *MessageBuilder {
	b.input.SetMessage(msg)
	return b
}

// WithSubject attaches a subject to the message.
func (b *MessageBuilder) WithSubject(subject string) *MessageBuilder {
	b.input.SetSubject(subject)
	return b
}

// TopicArn sets the topic ARN where the message will be sent.
func (b *MessageBuilder) TopicArn(topicArn string) *MessageBuilder {
	b.input.SetTopicArn(topicArn)
	return b
}

// TargetArn sets the target ARN where the message will be sent.
func (b *MessageBuilder) TargetArn(targetArn string) *MessageBuilder {
	b.input.SetTargetArn(targetArn)
	return b
}

// PhoneNumber sets the phone number to whom the message will be sent.
func (b *MessageBuilder) PhoneNumber(phoneNumber string) *MessageBuilder {
	b.input.SetPhoneNumber(phoneNumber)
	return b
}

// MessageStructure sets the message structure of the message.
func (b *MessageBuilder) MessageStructure(msgStructure string) *MessageBuilder {
	b.input.SetMessageStructure(msgStructure)
	return b
}

// WithStringAttribute attaches a string attribute to the message.
func (b *MessageBuilder) WithStringAttribute(name string, value string) *MessageBuilder {
	attributeValue := b.addAttributeValue(name, attributeDataTypeString)
	attributeValue.SetStringValue(value)
	return b
}

// WithStringArrayAttribute attaches an array of strings attributes to the message.
//
// The accepted values types are string, number, boolean and nil. Any other type will throw an error.
func (b *MessageBuilder) WithStringArrayAttribute(name string, values []interface{}) *MessageBuilder {
	attributeValue := b.addAttributeValue(name, attributeDataTypeStringArray)

	strValue, err := b.formatStringArrayAttributeValues(values)
	if err != nil {
		b.err = err
		return b
	}
	attributeValue.SetStringValue(strValue)

	return b
}

// formatStringArrayAttributeValues tries to format a slice of values that are used for string array attributes.
// It checks for specific, supported data types and returns a formatted string if data types are OK. It returns an
// error otherwise.
func (b *MessageBuilder) formatStringArrayAttributeValues(values []interface{}) (string, error) {
	for _, value := range values {
		switch t := value.(type) {
		case string, int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64, bool, nil:
			continue
		default:
			return "", fmt.Errorf("invalid string array attribute data type %T", t)
		}
	}

	strValue, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("could not create the string array attribute")
	}

	return string(strValue), nil
}

// WithNumberAttribute attaches a number attribute to the message, formatted as a string.
func (b *MessageBuilder) WithNumberAttribute(name string, value string) *MessageBuilder {
	attributeValue := b.addAttributeValue(name, attributeDataTypeNumber)
	attributeValue.SetStringValue(value)
	return b
}

// WithBinaryAttribute attaches a binary attribute to the message.
func (b *MessageBuilder) WithBinaryAttribute(name string, value []byte) *MessageBuilder {
	attributeValue := b.addAttributeValue(name, attributeDataTypeBinary)
	attributeValue.SetBinaryValue(value)
	return b
}

// addAttributeValue creates a base attribute value and adds it to the list of attribute values.
func (b *MessageBuilder) addAttributeValue(name string, dataType attributeDataType) *sns.MessageAttributeValue {
	attributeValue := &sns.MessageAttributeValue{}
	attributeValue.SetDataType(string(dataType))
	b.input.MessageAttributes[name] = attributeValue
	return attributeValue
}

// Build tries to build a message given its specified data and returns an error if any goes wrong.
func (b *MessageBuilder) Build() (*Message, error) {
	if b.err != nil {
		return nil, b.err
	}

	for name, attributeValue := range b.input.MessageAttributes {
		if err := attributeValue.Validate(); err != nil {
			return nil, fmt.Errorf("invalid attribute %s: %w", name, err)
		}
	}

	return &Message{input: b.input}, nil
}

// injectHeaders injects the SNS headers carrier's headers into the message's attributes.
func (m *Message) injectHeaders(carrier snsHeadersCarrier) {
	for k, v := range carrier {
		m.setMessageAttribute(k, v.(string)) //nolint:forcetypeassert
	}
}

func (m *Message) setMessageAttribute(key, value string) {
	m.input.MessageAttributes[key] = &sns.MessageAttributeValue{
		DataType:    aws.String(string(attributeDataTypeString)),
		StringValue: aws.String(value),
	}
}
