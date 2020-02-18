package sqs

import (
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_MessageBuilder_Build(t *testing.T) {
	b := NewMessageBuilder()

	body := "body"
	queueURL := "url"
	delay := int64(10)

	stringAttribute := "string attribute"
	numberAttribute := int64(1337)
	binaryAttribute := []byte("binary attribute")

	got, err := b.
		Body(body).
		QueueURL(queueURL).
		WithDelaySeconds(delay).
		WithStringAttribute("string", stringAttribute).
		WithNumberAttribute("number", numberAttribute).
		WithBinaryAttribute("binary", binaryAttribute).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, got.input.MessageBody, &body)
	assert.Equal(t, got.input.QueueUrl, &queueURL)
	assert.Equal(t, got.input.DelaySeconds, &delay)

	assert.Equal(t, string(attributeDataTypeString), *got.input.MessageAttributes["string"].DataType)
	assert.Equal(t, stringAttribute, *got.input.MessageAttributes["string"].StringValue)

	assert.Equal(t, string(attributeDataTypeNumber), *got.input.MessageAttributes["number"].DataType)
	assert.Equal(t, strconv.FormatInt(numberAttribute, 10), *got.input.MessageAttributes["number"].StringValue)

	assert.Equal(t, string(attributeDataTypeBinary), *got.input.MessageAttributes["binary"].DataType)
	assert.Equal(t, binaryAttribute, got.input.MessageAttributes["binary"].BinaryValue)
}

func Test_MessageBuilder_Build_Fifo(t *testing.T) {
	b := NewMessageBuilder()

	body := "body"
	queueURL := "url"
	deduplicationID := "deduplication ID"
	groupID := "group ID"

	got, err := b.
		Body(body).
		QueueURL(queueURL).
		WithDeduplicationID(deduplicationID).
		WithGroupID(groupID).
		Build()

	assert.NoError(t, err)
	assert.Equal(t, got.input.MessageBody, &body)
	assert.Equal(t, got.input.QueueUrl, &queueURL)
	assert.Equal(t, got.input.MessageDeduplicationId, &deduplicationID)
	assert.Equal(t, got.input.MessageGroupId, &groupID)
}

func Test_MessageBuilder_Build_With_Error(t *testing.T) {
	b := NewMessageBuilder()
	err := errors.New("an err")
	b.err = err
	m, foundErr := b.Build()
	assert.Nil(t, m)
	assert.EqualError(t, err, foundErr.Error())

	testCases := map[string]struct {
		msgBuilder  *MessageBuilder
		expectedErr error
	}{
		"missing body": {
			msgBuilder:  NewMessageBuilder(),
			expectedErr: errors.New("missing required field: message body"),
		},
		"missing queue URL": {
			msgBuilder:  NewMessageBuilder().Body("body"),
			expectedErr: errors.New("missing required field: message queue URL"),
		},
		"group ID and delay": {
			msgBuilder: NewMessageBuilder().Body("body").QueueURL("url").
				WithGroupID("id").WithDelaySeconds(1),
			expectedErr: errors.New("could not set a delay with either a group ID or a deduplication ID"),
		},
		"deduplication ID and delay": {
			msgBuilder: NewMessageBuilder().Body("body").QueueURL("url").
				WithDeduplicationID("id").WithDelaySeconds(1),
			expectedErr: errors.New("could not set a delay with either a group ID or a deduplication ID"),
		},
		"group ID and deduplication ID and delay": {
			msgBuilder: NewMessageBuilder().Body("body").QueueURL("url").
				WithGroupID("id").WithDeduplicationID("id").WithDelaySeconds(1),
			expectedErr: errors.New("could not set a delay with either a group ID or a deduplication ID"),
		},
	}
	for name, tC := range testCases {
		t.Run(name, func(t *testing.T) {
			msg, err := tC.msgBuilder.Build()
			if tC.expectedErr != nil {
				assert.Nil(t, m)
				assert.EqualError(t, err, tC.expectedErr.Error())
			} else {
				assert.IsType(t, &Message{}, msg)
				assert.NoError(t, err)
			}
		})
	}
}

func TestMessage_injectHeaders(t *testing.T) {
	msg, err := NewMessageBuilder().Body("body").QueueURL("url").Build()
	require.NoError(t, err)

	carrier := sqsHeadersCarrier{
		"foo": "bar",
		"bar": "baz",
	}
	msg.injectHeaders(carrier)

	assert.Equal(t, "bar", *msg.input.MessageAttributes["foo"].StringValue)
	assert.Equal(t, "baz", *msg.input.MessageAttributes["bar"].StringValue)
}
