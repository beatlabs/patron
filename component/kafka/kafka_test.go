package kafka

import (
	"context"
	"testing"

	"github.com/beatlabs/patron/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

func Test_messageWrapper(t *testing.T) {
	rec := &kgo.Record{
		Headers: []kgo.RecordHeader{
			{
				Key:   correlation.HeaderID,
				Value: []byte("18914117-d9c9-4d0f-941c-d0efbb25fb45"),
			},
		},
		Topic: "topicone",
		Value: []byte(`{"key":"value"}`),
	}
	ctx := context.Background()
	msg := message{
		ctx: ctx,
		rec: rec,
	}

	msgCtx := msg.Context()
	consumerRecord := msg.Record()
	assert.Equal(t, ctx, msgCtx)
	assert.NotNil(t, consumerRecord)
	assert.Equal(t, "topicone", consumerRecord.Topic)
	assert.JSONEq(t, `{"key":"value"}`, string(consumerRecord.Value))
}

func Test_NewBatch(t *testing.T) {
	ctx := context.Background()
	rec := &kgo.Record{
		Headers: []kgo.RecordHeader{
			{
				Key:   correlation.HeaderID,
				Value: []byte("18914117-d9c9-4d0f-941c-d0efbb25fb45"),
			},
		},
		Topic: "topicone",
		Value: []byte(`{"key":"value"}`),
	}

	msg := NewMessage(ctx, nil, rec)
	btc := NewBatch([]Message{msg})
	assert.Len(t, btc.Messages(), 1)
}

func Test_Message(t *testing.T) {
	ctx := context.Background()
	rec := &kgo.Record{
		Headers: []kgo.RecordHeader{
			{
				Key:   correlation.HeaderID,
				Value: []byte("18914117-d9c9-4d0f-941c-d0efbb25fb45"),
			},
		},
		Topic: "topicone",
		Value: []byte(`{"key":"value"}`),
	}

	msg := NewMessage(ctx, nil, rec)
	assert.Equal(t, ctx, msg.Context())
	assert.Nil(t, msg.Span())
	assert.Equal(t, rec, msg.Record())
}

func Test_DefaultConsumerConfig(t *testing.T) {
	opts, err := DefaultConsumerConfig("name", true)
	require.NoError(t, err)
	require.NotEmpty(t, opts)

	opts, err = DefaultConsumerConfig("name", false)
	require.NoError(t, err)
	require.NotEmpty(t, opts)
}
