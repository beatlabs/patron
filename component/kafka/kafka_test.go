package kafka

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/require"

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

	span := mocktracer.New().StartSpan("msg")
	msg := NewMessage(ctx, span, cm)
	btc := NewBatch([]Message{msg})
	assert.Equal(t, 1, len(btc.Messages()))
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

	span := mocktracer.New().StartSpan("msg")
	msg := NewMessage(ctx, span, cm)
	assert.Equal(t, ctx, msg.Context())
	assert.Equal(t, span, msg.Span())
	assert.Equal(t, cm, msg.Message())
}

func Test_DefaultConsumerSaramaConfig(t *testing.T) {
	sc, err := DefaultConsumerSaramaConfig("name", true)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(sc.ClientID, fmt.Sprintf("-%s", "name")))
	require.Equal(t, sarama.ReadCommitted, sc.Consumer.IsolationLevel)

	sc, err = DefaultConsumerSaramaConfig("name", false)
	require.NoError(t, err)
	require.NotEqual(t, sarama.ReadCommitted, sc.Consumer.IsolationLevel)
}

func TestDeduplicateMessages(t *testing.T) {
	message := func(key, val string) Message {
		return NewMessage(
			context.TODO(),
			opentracing.SpanFromContext(context.TODO()),
			&sarama.ConsumerMessage{Key: []byte(key), Value: []byte(val)})
	}
	find := func(collection []Message, key string) Message {
		for _, m := range collection {
			if string(m.Message().Key) == key {
				return m
			}
		}
		return nil
	}

	// Given
	original := []Message{
		message("k1", "v1.1"),
		message("k1", "v1.2"),
		message("k2", "v2.1"),
		message("k2", "v2.2"),
		message("k1", "v1.3"),
	}

	cleaned := DeduplicateMessages(original)

	assert.Len(t, cleaned, 2)
	assert.Equal(t, []byte("v1.3"), find(cleaned, "k1").Message().Value)
	assert.Equal(t, []byte("v2.2"), find(cleaned, "k2").Message().Value)
}
