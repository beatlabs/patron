package sns

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/service/sns"
)

type attributeDataType string

const (
	attributeDataTypeString      attributeDataType = "String"
	attributeDataTypeStringArray attributeDataType = "StringArray"
	attributeDataTypeNumber      attributeDataType = "Number"
	attributeDataTypeBinary      attributeDataType = "Binary"
)

// Message stores information about messages that are published to SNS thanks to the SNS publisher.
type Message struct {
	input *sns.PublishInput
}

// MessageBuilder helps building messages.
type MessageBuilder struct {
	err   error
	input *sns.PublishInput
}

// NewMessageBuilder creates a new MessageBuilder that helps creating messages.
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		input: &sns.PublishInput{
			MessageAttributes: map[string]*sns.MessageAttributeValue{},
		},
	}
}

// WithMessage attaches a message to the message struct.
func (b *MessageBuilder) WithMessage(msg string) *MessageBuilder {
	b.input.SetMessage(msg)
	return b
}

// WithSubject attaches a subject to the message.
func (b *MessageBuilder) WithSubject(subject string) *MessageBuilder {
	b.input.SetSubject(subject)
	return b
}

// WithTopicARN sets the topic ARN where the message will be sent.
func (b *MessageBuilder) WithTopicARN(topicARN string) *MessageBuilder {
	b.input.SetTopicArn(topicARN)
	return b
}

// WithTargetARN sets the target ARN where the message will be sent.
func (b *MessageBuilder) WithTargetARN(targetARN string) *MessageBuilder {
	b.input.SetTargetArn(targetARN)
	return b
}

// WithPhoneNumber sets the phone number to whom the message will be sent.
func (b *MessageBuilder) WithPhoneNumber(phoneNumber string) *MessageBuilder {
	b.input.SetPhoneNumber(phoneNumber)
	return b
}

// WithMessageStructure sets the message structure of the message.
func (b *MessageBuilder) WithMessageStructure(msgStructure string) *MessageBuilder {
	b.input.SetMessageStructure(msgStructure)
	return b
}

// WithStringAttribute attaches a string attribute to the message.
func (b *MessageBuilder) WithStringAttribute(name string, value string) *MessageBuilder {
	attributeValue := b.addAttributeValue(name, attributeDataTypeString)
	attributeValue.SetStringValue(value)
	return b
}

// WithStringArrayAttribute attaches an array of strings attribute to the message.
// Accepted values types: string, numbers, booleans and nil. Any other type will throw an error.
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

// addAttributeValue creates a base attribute value and adds it to the the list of attribute values.
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
			return nil, fmt.Errorf("invalid attribute %s: %v", name, err)
		}
	}

	return &Message{input: b.input}, nil
}
