package sns

import (
	"errors"
	"testing"

	sns "github.com/aws/aws-sdk-go/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_MessageBuilder_Build(t *testing.T) {
	b := NewMessageBuilder()

	msg := "message"
	subject := "subject"
	topicArn := "topic ARN"
	targetArn := "target ARN"
	phoneNumber := "phone number"
	msgStructure := "msg structure"
	stringAttribute := "string attribute"
	stringArrayAttribute := []interface{}{"foo", "bar"}
	numberAttribute := "13.37"
	binaryAttribute := []byte("binary attribute")

	got, err := b.
		Message(msg).
		WithSubject(subject).
		TopicArn(topicArn).
		TargetArn(targetArn).
		PhoneNumber(phoneNumber).
		MessageStructure(msgStructure).
		WithStringAttribute("string", stringAttribute).
		WithStringArrayAttribute("string_array", stringArrayAttribute).
		WithNumberAttribute("number", numberAttribute).
		WithBinaryAttribute("binary", binaryAttribute).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, got.input.Message, &msg)
	assert.Equal(t, got.input.Subject, &subject)
	assert.Equal(t, got.input.TopicArn, &topicArn)
	assert.Equal(t, got.input.TargetArn, &targetArn)
	assert.Equal(t, got.input.PhoneNumber, &phoneNumber)
	assert.Equal(t, got.input.MessageStructure, &msgStructure)

	assert.Equal(t, string(attributeDataTypeString), *got.input.MessageAttributes["string"].DataType)
	assert.Equal(t, stringAttribute, *got.input.MessageAttributes["string"].StringValue)

	assert.Equal(t, string(attributeDataTypeStringArray), *got.input.MessageAttributes["string_array"].DataType)
	assert.Equal(t, `["foo","bar"]`, *got.input.MessageAttributes["string_array"].StringValue)

	assert.Equal(t, string(attributeDataTypeNumber), *got.input.MessageAttributes["number"].DataType)
	assert.Equal(t, numberAttribute, *got.input.MessageAttributes["number"].StringValue)

	assert.Equal(t, string(attributeDataTypeBinary), *got.input.MessageAttributes["binary"].DataType)
	assert.Equal(t, binaryAttribute, got.input.MessageAttributes["binary"].BinaryValue)
}

func Test_MessageBuilder_InvalidStringArrayAttribute(t *testing.T) {
	b := NewMessageBuilder()
	b.WithStringArrayAttribute("attr", []interface{}{struct{}{}})

	assert.Equal(t, errors.New("invalid string array attribute data type struct {}"), b.err)
}

func Test_MessageBuilder_Build_With_Error(t *testing.T) {
	b := NewMessageBuilder()
	errMsg := "an err"
	b.err = errors.New(errMsg)
	m, foundErr := b.Build()
	assert.Nil(t, m)
	assert.EqualError(t, foundErr, errMsg)
}

func Test_MessageBuilder_Build_With_Invalid_attribute(t *testing.T) {
	b := NewMessageBuilder()
	b.input.MessageAttributes["attr"] = &sns.MessageAttributeValue{}
	msg, err := b.Build()
	assert.Nil(t, msg)
	assert.Error(t, err)
}

func Test_MessageBuilder_formatStringArrayAttributeValues(t *testing.T) {
	testCases := []struct {
		desc           string
		values         []interface{}
		expectedOutput string
		expectedErr    error
	}{
		{
			desc: "Valid data types - (u)ints",
			values: []interface{}{
				42, 42, int8(42), int16(42), int32(42), int64(42),
				uint(42), uint8(42), uint16(42), uint32(42), uint64(42),
			},
			expectedOutput: `[42,42,42,42,42,42,42,42,42,42,42]`,
		},
		{
			desc:           "Valid data types - floats",
			values:         []interface{}{float32(13.37), 13.37, 13.37},
			expectedOutput: `[13.37,13.37,13.37]`,
		},
		{
			desc:           "Valid data types - rest",
			values:         []interface{}{"foo", true, false, nil},
			expectedOutput: `["foo",true,false,null]`,
		},
		{
			desc:        "Invalid - struct",
			values:      []interface{}{struct{}{}},
			expectedErr: errors.New("invalid string array attribute data type struct {}"),
		},
		{
			desc:        "Invalid - slice",
			values:      []interface{}{[]interface{}{}},
			expectedErr: errors.New("invalid string array attribute data type []interface {}"),
		},
		{
			desc:        "Invalid - func",
			values:      []interface{}{func() {}},
			expectedErr: errors.New("invalid string array attribute data type func()"),
		},
		{
			desc:        "Invalid - chan",
			values:      []interface{}{make(chan int)},
			expectedErr: errors.New("invalid string array attribute data type chan int"),
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			b := NewMessageBuilder()
			got, err := b.formatStringArrayAttributeValues(tC.values)

			assert.Equal(t, tC.expectedOutput, got)

			if tC.expectedErr != nil {
				assert.EqualError(t, err, tC.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_Message_tracingTarget(t *testing.T) {
	msgWithTopicArn, err := NewMessageBuilder().TopicArn("topic-arn").Build()
	require.NoError(t, err)

	msgWithTargetArn, err := NewMessageBuilder().TargetArn("target-arn").Build()
	require.NoError(t, err)

	msgWithPhoneNumber, err := NewMessageBuilder().PhoneNumber("my-phone-number").Build()
	require.NoError(t, err)

	blankMsg, err := NewMessageBuilder().Build()
	require.NoError(t, err)

	testCases := []struct {
		desc                  string
		msg                   *Message
		expectedTracingTarget string
	}{
		{
			desc:                  "Topic ARN",
			msg:                   msgWithTopicArn,
			expectedTracingTarget: "topic-arn:topic-arn",
		},
		{
			desc:                  "Target ARN",
			msg:                   msgWithTargetArn,
			expectedTracingTarget: "target-arn:target-arn",
		},
		{
			desc:                  "Phone number",
			msg:                   msgWithPhoneNumber,
			expectedTracingTarget: "phone-number",
		},
		{
			desc:                  "Unknown",
			msg:                   blankMsg,
			expectedTracingTarget: "unknown",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			got := tC.msg.tracingTarget()
			assert.Equal(t, tC.expectedTracingTarget, got)
		})
	}
}

func TestMessage_injectHeaders(t *testing.T) {
	msg, err := NewMessageBuilder().Build()
	require.NoError(t, err)

	carrier := snsHeadersCarrier{
		"foo": "bar",
		"bar": "baz",
	}
	msg.injectHeaders(carrier)

	assert.Equal(t, "bar", *msg.input.MessageAttributes["foo"].StringValue)
	assert.Equal(t, "baz", *msg.input.MessageAttributes["bar"].StringValue)
}
