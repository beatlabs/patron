package kafka

import (
	"testing"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/correlation"
	"github.com/stretchr/testify/assert"
)

func Test_messageWrapper(t *testing.T) {
	cm := &sarama.ConsumerMessage{
		Headers: []*sarama.RecordHeader{
			{
				Key:   []byte(correlation.HeaderID),
				Value: []byte("18914117-d9c9-4d0f-941c-d0efbb25fb45"),
			},
		},
		Topic: "topicone",
		Value: []byte(`{"key":"value"}`),
	}
	msg := messageWrapper{
		msg: cm,
	}

	consumerMessage := msg.GetConsumerMessage()
	assert.NotNil(t, consumerMessage)
	assert.Equal(t, "topicone", consumerMessage.Topic)
	assert.Equal(t, []byte(`{"key":"value"}`), consumerMessage.Value)
	assert.Equal(t, "18914117-d9c9-4d0f-941c-d0efbb25fb45", msg.GetCorrelationID())

	// another message with no correlation ID
	cm = &sarama.ConsumerMessage{
		Topic: "topictwo",
		Value: []byte(`{"k":"v"}`),
	}
	msg = messageWrapper{
		msg: cm,
	}
	assert.NotNil(t, msg.GetCorrelationID()) // a new correlation ID is generated
}
