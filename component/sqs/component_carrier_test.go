package sqs

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const traceparentKey = "traceparent"

func TestConsumerMessageCarrierGet(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		message  types.Message
		key      string
		expected string
	}{
		"returns value from message attributes": {
			message: types.Message{MessageAttributes: map[string]types.MessageAttributeValue{
				traceparentKey: {StringValue: aws.String("00-abc-123-01")},
			}},
			key:      traceparentKey,
			expected: "00-abc-123-01",
		},
		"missing attribute returns empty string": {
			message:  types.Message{},
			key:      traceparentKey,
			expected: "",
		},
		"non string attribute returns empty string": {
			message: types.Message{MessageAttributes: map[string]types.MessageAttributeValue{
				traceparentKey: {DataType: aws.String("String")},
			}},
			key:      traceparentKey,
			expected: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			carrier := consumerMessageCarrier{msg: &tt.message}

			assert.Equal(t, tt.expected, carrier.Get(tt.key))
		})
	}
}

func TestConsumerMessageCarrierSet(t *testing.T) {
	t.Parallel()

	message := types.Message{}
	carrier := consumerMessageCarrier{msg: &message}

	carrier.Set(traceparentKey, "00-abc-123-01")

	require.Contains(t, message.MessageAttributes, traceparentKey)
	assert.Equal(t, "String", aws.ToString(message.MessageAttributes[traceparentKey].DataType))
	assert.Equal(t, "00-abc-123-01", aws.ToString(message.MessageAttributes[traceparentKey].StringValue))
}

func TestConsumerMessageCarrierKeys(t *testing.T) {
	t.Parallel()

	message := types.Message{MessageAttributes: map[string]types.MessageAttributeValue{
		traceparentKey: {StringValue: aws.String("00-abc-123-01")},
		"baggage":      {StringValue: aws.String("k=v")},
	}}
	carrier := consumerMessageCarrier{msg: &message}

	assert.ElementsMatch(t, []string{traceparentKey, "baggage"}, carrier.Keys())
}
