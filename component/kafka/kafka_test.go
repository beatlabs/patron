package kafka

import (
	"context"
	"fmt"
	"strings"
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
	ctx := context.Background()
	msg := message{
		ctx: ctx,
		msg: cm,
	}

	msgCtx := msg.Context()
	consumerMessage := msg.Message()
	assert.Equal(t, ctx, msgCtx)
	assert.NotNil(t, consumerMessage)
	assert.Equal(t, "topicone", consumerMessage.Topic)
	assert.Equal(t, []byte(`{"key":"value"}`), consumerMessage.Value)
}

func Test_defaultSaramaConfig(t *testing.T) {
	sc, err := defaultSaramaConfig("name")
	assert.NoError(t, err)
	assert.True(t, strings.HasSuffix(sc.ClientID, fmt.Sprintf("-%s", "name")))
}
