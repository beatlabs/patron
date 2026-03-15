package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("nil brokers", func(t *testing.T) {
		t.Parallel()
		got, err := New(nil)
		require.EqualError(t, err, "brokers are empty or have an empty value")
		require.Nil(t, got)
	})

	t.Run("empty brokers", func(t *testing.T) {
		t.Parallel()
		got, err := New([]string{})
		require.EqualError(t, err, "brokers are empty or have an empty value")
		require.Nil(t, got)
	})

	t.Run("empty value", func(t *testing.T) {
		t.Parallel()
		got, err := New([]string{""})
		require.EqualError(t, err, "brokers are empty or have an empty value")
		require.Nil(t, got)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		got, err := New([]string{"localhost:9092"})
		require.NoError(t, err)
		require.NotNil(t, got)
		got.Close()
	})
}

func TestSendAsync_Failure(t *testing.T) {
	t.Parallel()

	p, err := New([]string{"localhost:19092"})
	require.NoError(t, err)
	p.client.Close()

	chErr := make(chan error, 1)
	rec := &kgo.Record{Topic: "test-topic", Value: []byte("test")}
	p.SendAsync(context.Background(), rec, chErr)

	select {
	case err := <-chErr:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send message")
	case <-time.After(5 * time.Second):
		t.Fatal("expected error on chErr but timed out")
	}
}

func TestSend_Validation(t *testing.T) {
	t.Parallel()

	t.Run("empty records returns error", func(t *testing.T) {
		t.Parallel()

		p := &Producer{}

		_, err := p.Send(context.Background())

		require.EqualError(t, err, "records are empty or nil")
	})

	t.Run("nil records returns error", func(t *testing.T) {
		t.Parallel()

		p := &Producer{}

		_, err := p.Send(context.Background(), []*kgo.Record{}...)

		require.EqualError(t, err, "records are empty or nil")
	})
}

func TestInjectCorrelationHeader(t *testing.T) {
	t.Parallel()

	t.Run("injects correlation ID", func(t *testing.T) {
		t.Parallel()

		ctx := correlation.ContextWithID(context.Background(), "123")
		rec := &kgo.Record{Topic: "topic"}

		injectCorrelationHeader(ctx, rec)

		require.Len(t, rec.Headers, 1)
		assert.Equal(t, correlation.HeaderID, rec.Headers[0].Key)
		assert.Equal(t, "123", string(rec.Headers[0].Value))
	})

	t.Run("handles nil headers", func(t *testing.T) {
		t.Parallel()

		ctx := correlation.ContextWithID(context.Background(), "test-123")
		rec := &kgo.Record{
			Topic:   "test",
			Headers: nil,
		}

		injectCorrelationHeader(ctx, rec)

		require.Len(t, rec.Headers, 1)
		assert.Equal(t, correlation.HeaderID, rec.Headers[0].Key)
		assert.Equal(t, "test-123", string(rec.Headers[0].Value))
	})

	t.Run("preserves existing headers", func(t *testing.T) {
		t.Parallel()

		ctx := correlation.ContextWithID(context.Background(), "test-456")
		rec := &kgo.Record{
			Topic: "test",
			Headers: []kgo.RecordHeader{
				{Key: "existing", Value: []byte("header")},
			},
		}

		injectCorrelationHeader(ctx, rec)

		require.Len(t, rec.Headers, 2)
		assert.Equal(t, "existing", rec.Headers[0].Key)
		assert.Equal(t, "header", string(rec.Headers[0].Value))
		assert.Equal(t, correlation.HeaderID, rec.Headers[1].Key)
		assert.Equal(t, "test-456", string(rec.Headers[1].Value))
	})
}

func TestTopicAttribute(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		topic    string
		expected string
	}{
		"simple topic":       {topic: "test-topic", expected: "test-topic"},
		"dotted topic":       {topic: "test.topic.name", expected: "test.topic.name"},
		"empty topic":        {topic: "", expected: ""},
		"special characters": {topic: "test_topic-123", expected: "test_topic-123"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			attr := topicAttribute(tt.topic)

			assert.Equal(t, "topic", string(attr.Key))
			assert.Equal(t, tt.expected, attr.Value.AsString())
		})
	}
}

func TestPublishCountAdd(t *testing.T) {
	t.Parallel()

	t.Run("does not panic with valid context", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			publishCountAdd(context.Background(), topicAttribute("test-topic"))
		})
	})

	t.Run("does not panic with cancelled context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		assert.NotPanics(t, func() {
			publishCountAdd(ctx, topicAttribute("test-topic"))
		})
	})
}

func TestDeliveryTypeConstants(t *testing.T) {
	t.Parallel()

	t.Run("sync delivery type", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "sync", deliveryTypeSync)
		assert.Equal(t, "delivery", string(deliveryTypeSyncAttr.Key))
		assert.Equal(t, "sync", deliveryTypeSyncAttr.Value.AsString())
	})

	t.Run("async delivery type", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "async", deliveryTypeAsync)
		assert.Equal(t, "delivery", string(deliveryTypeAsyncAttr.Key))
		assert.Equal(t, "async", deliveryTypeAsyncAttr.Value.AsString())
	})
}
