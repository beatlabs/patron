package sqs

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type attributeDataType string

const (
	attributeDataTypeString attributeDataType = "String"
	attributeDataTypeNumber attributeDataType = "Number"
	attributeDataTypeBinary attributeDataType = "Binary"
	attributeDataTypeCustom attributeDataType = "Custom"
)

var customDataTypeRegex map[attributeDataType]*regexp.Regexp

func init() {
	customDataTypeRegex = make(map[attributeDataType]*regexp.Regexp, 3)
	customDataTypeRegex[attributeDataTypeString] = regexp.MustCompile(`String\.\w+`)
	customDataTypeRegex[attributeDataTypeNumber] = regexp.MustCompile(`Number\.\w+`)
	customDataTypeRegex[attributeDataTypeBinary] = regexp.MustCompile(`Binary\.\w+`)
}

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
func (b *MessageBuilder) WithCustomStringAttribute(name, value, customDataType string) *MessageBuilder {
	ok := b.validateCustomAttributeDataType(customDataType, attributeDataTypeString)
	if !ok {
		b.err = errors.New("invalid custom attribute data type format, check the AWS SQS doc")
	}
	v := b.addAttributeValue(name, attributeDataTypeString)
	v.SetStringValue(value)
	return b
}

func (b *MessageBuilder) WithCustomNumberAttribute(name string, value int64, customDataType string) *MessageBuilder {
	ok := b.validateCustomAttributeDataType(customDataType, attributeDataTypeNumber)
	if !ok {
		b.err = errors.New("invalid custom attribute data type format, check the AWS SQS doc")
	}
	v := b.addAttributeValue(name, attributeDataTypeNumber)
	v.SetStringValue(strconv.FormatInt(value, 10)) // We need to use StringValue for Number (see SDK doc)
	return b
}

func (b *MessageBuilder) WithCustomBinaryAttribute(name string, value []byte, customDataType string) *MessageBuilder {
	ok := b.validateCustomAttributeDataType(customDataType, attributeDataTypeBinary)
	if !ok {
		b.err = errors.New("invalid custom attribute data type format, check theAWS SQS doc")
	}
	v := b.addAttributeValue(name, attributeDataTypeBinary)
	v.SetBinaryValue(value)
	return b
}

func (b *MessageBuilder) validateCustomAttributeDataType(customDataType string, origDataType attributeDataType) bool {
	r, ok := customDataTypeRegex[origDataType]
	if !ok {
		return false
	}
	return r.MatchString(customDataType)
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
