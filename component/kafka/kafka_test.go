package kafka

import (
	"context"
	"strings"
	"testing"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.JSONEq(t, `{"key":"value"}`, string(consumerMessage.Value))
}

func Test_NewBatch(t *testing.T) {
	ctx := context.Background()
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

	msg := NewMessage(ctx, nil, cm)
	btc := NewBatch([]Message{msg})
	assert.Len(t, btc.Messages(), 1)
}

func Test_Message(t *testing.T) {
	ctx := context.Background()
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

	msg := NewMessage(ctx, nil, cm)
	assert.Equal(t, ctx, msg.Context())
	assert.Nil(t, msg.Span())
	assert.Equal(t, cm, msg.Message())
}

func Test_DefaultConsumerSaramaConfig(t *testing.T) {
	sc, err := DefaultConsumerSaramaConfig("name", true)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(sc.ClientID, "-name"))
	require.Equal(t, sarama.ReadCommitted, sc.Consumer.IsolationLevel)

	sc, err = DefaultConsumerSaramaConfig("name", false)
	require.NoError(t, err)
	require.NotEqual(t, sarama.ReadCommitted, sc.Consumer.IsolationLevel)
}
