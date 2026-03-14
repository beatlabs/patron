package kafka

import (
	"context"
	"testing"

	"github.com/beatlabs/patron/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		brokers     []string
		expectedErr string
	}{
		"nil brokers":   {brokers: nil, expectedErr: "brokers are empty or have an empty value"},
		"empty brokers": {brokers: []string{}, expectedErr: "brokers are empty or have an empty value"},
		"empty value":   {brokers: []string{""}, expectedErr: "brokers are empty or have an empty value"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := New(tt.brokers)

			require.EqualError(t, err, tt.expectedErr)
			require.Nil(t, got)
		})
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
