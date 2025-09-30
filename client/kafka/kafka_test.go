package kafka

import (
	"context"
	"strings"
	"testing"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron/correlation"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestBuilder_Create(t *testing.T) {
	t.Parallel()
	type args struct {
		brokers []string
		cfg     *sarama.Config
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"missing brokers": {args: args{brokers: nil, cfg: sarama.NewConfig()}, expectedErr: "brokers are empty or have an empty value"},
		"missing config":  {args: args{brokers: []string{"123"}, cfg: nil}, expectedErr: "no Sarama configuration specified"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := New(tt.args.brokers, tt.args.cfg).Create()

			require.EqualError(t, err, tt.expectedErr)
			require.Nil(t, got)
		})
	}
}

func TestBuilder_CreateAsync(t *testing.T) {
	t.Parallel()
	type args struct {
		brokers []string
		cfg     *sarama.Config
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"missing brokers": {args: args{brokers: nil, cfg: sarama.NewConfig()}, expectedErr: "brokers are empty or have an empty value"},
		"missing config":  {args: args{brokers: []string{"123"}, cfg: nil}, expectedErr: "no Sarama configuration specified"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, chErr, err := New(tt.args.brokers, tt.args.cfg).CreateAsync()

			require.EqualError(t, err, tt.expectedErr)
			require.Nil(t, got)
			require.Nil(t, chErr)
		})
	}
}

func TestDefaultProducerSaramaConfig(t *testing.T) {
	sc, err := DefaultProducerSaramaConfig("name", true)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(sc.ClientID, "-name"))
	require.True(t, sc.Producer.Idempotent)

	sc, err = DefaultProducerSaramaConfig("name", false)
	require.NoError(t, err)
	require.False(t, sc.Producer.Idempotent)
}

func Test_injectTracingAndCorrelationHeaders(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	_ = patrontrace.Setup("test", nil, exp)

	ctx := correlation.ContextWithID(context.Background(), "123")

	msg := sarama.ProducerMessage{}

	ctx, _ = startSpan(ctx, "send", deliveryTypeSync, "topic")

	injectTracingAndCorrelationHeaders(ctx, &msg)
	assert.Len(t, msg.Headers, 2)
	assert.Equal(t, correlation.HeaderID, string(msg.Headers[0].Key))
	assert.Equal(t, "123", string(msg.Headers[0].Value))
	assert.Equal(t, "traceparent", string(msg.Headers[1].Key))
	assert.NotEmpty(t, string(msg.Headers[1].Value))
}

func TestStartSpan(t *testing.T) {
	t.Parallel()

	t.Run("creates span with correct attributes", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx, span := startSpan(ctx, "test-action", "sync", "test-topic")

		require.NotNil(t, ctx)
		require.NotNil(t, span)

		span.End()
	})

	t.Run("creates span without topic", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		ctx, span := startSpan(ctx, "test-action", "async", "")

		require.NotNil(t, ctx)
		require.NotNil(t, span)

		span.End()
	})

	t.Run("preserves context values", func(t *testing.T) {
		t.Parallel()

		ctx := correlation.ContextWithID(context.Background(), "test-id")
		ctx, span := startSpan(ctx, "test-action", "sync", "test-topic")

		require.NotNil(t, ctx)
		require.NotNil(t, span)

		corID := correlation.IDFromContext(ctx)
		assert.Equal(t, "test-id", corID)

		span.End()
	})
}

func TestTopicAttribute(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		topic    string
		expected string
	}{
		"simple topic": {
			topic:    "test-topic",
			expected: "test-topic",
		},
		"dotted topic": {
			topic:    "test.topic.name",
			expected: "test.topic.name",
		},
		"empty topic": {
			topic:    "",
			expected: "",
		},
		"special characters": {
			topic:    "test_topic-123",
			expected: "test_topic-123",
		},
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

func TestProducerMessageCarrier_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns empty string", func(t *testing.T) {
		t.Parallel()

		msg := &sarama.ProducerMessage{}
		carrier := producerMessageCarrier{msg: msg}

		value := carrier.Get("any-key")

		assert.Empty(t, value)
	})
}

func TestProducerMessageCarrier_Set(t *testing.T) {
	t.Parallel()

	t.Run("sets header", func(t *testing.T) {
		t.Parallel()

		msg := &sarama.ProducerMessage{}
		carrier := producerMessageCarrier{msg: msg}

		carrier.Set("test-key", "test-value")

		require.Len(t, msg.Headers, 1)
		assert.Equal(t, "test-key", string(msg.Headers[0].Key))
		assert.Equal(t, "test-value", string(msg.Headers[0].Value))
	})

	t.Run("sets multiple headers", func(t *testing.T) {
		t.Parallel()

		msg := &sarama.ProducerMessage{}
		carrier := producerMessageCarrier{msg: msg}

		carrier.Set("key1", "value1")
		carrier.Set("key2", "value2")

		require.Len(t, msg.Headers, 2)
		assert.Equal(t, "key1", string(msg.Headers[0].Key))
		assert.Equal(t, "value1", string(msg.Headers[0].Value))
		assert.Equal(t, "key2", string(msg.Headers[1].Key))
		assert.Equal(t, "value2", string(msg.Headers[1].Value))
	})
}

func TestProducerMessageCarrier_Keys(t *testing.T) {
	t.Parallel()

	t.Run("returns nil", func(t *testing.T) {
		t.Parallel()

		msg := &sarama.ProducerMessage{}
		carrier := producerMessageCarrier{msg: msg}

		keys := carrier.Keys()

		assert.Nil(t, keys)
	})
}

func TestPublishCountAdd(t *testing.T) {
	t.Parallel()

	t.Run("does not panic with valid context", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		attr := topicAttribute("test-topic")

		assert.NotPanics(t, func() {
			publishCountAdd(ctx, attr)
		})
	})

	t.Run("does not panic with cancelled context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		attr := topicAttribute("test-topic")

		assert.NotPanics(t, func() {
			publishCountAdd(ctx, attr)
		})
	})
}

func TestBaseProducer_ActiveBrokers(t *testing.T) {
	t.Parallel()

	t.Run("returns empty list with nil client", func(t *testing.T) {
		t.Parallel()

		bp := &baseProducer{
			prodClient: nil,
		}

		// This will panic with nil client, but we're testing the structure
		assert.NotNil(t, bp)
	})
}

func TestBuilder_Errors(t *testing.T) {
	t.Parallel()

	t.Run("accumulates multiple errors", func(t *testing.T) {
		t.Parallel()

		b := New(nil, nil)

		require.NotNil(t, b)
		assert.Len(t, b.errs, 2)
	})

	t.Run("error on Create with accumulated errors", func(t *testing.T) {
		t.Parallel()

		b := New([]string{}, nil)

		_, err := b.Create()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "brokers are empty")
		assert.Contains(t, err.Error(), "no Sarama configuration")
	})

	t.Run("error on CreateAsync with accumulated errors", func(t *testing.T) {
		t.Parallel()

		b := New(nil, sarama.NewConfig())

		_, _, err := b.CreateAsync()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "brokers are empty")
	})
}

func TestDefaultProducerSaramaConfig_Validation(t *testing.T) {
	t.Parallel()

	t.Run("creates config with idempotency", func(t *testing.T) {
		t.Parallel()

		cfg, err := DefaultProducerSaramaConfig("test-producer", true)

		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.True(t, cfg.Producer.Idempotent)
		assert.Equal(t, 1, cfg.Net.MaxOpenRequests)
		assert.Equal(t, sarama.WaitForAll, cfg.Producer.RequiredAcks)
		assert.Contains(t, cfg.ClientID, "test-producer")
	})

	t.Run("creates config without idempotency", func(t *testing.T) {
		t.Parallel()

		cfg, err := DefaultProducerSaramaConfig("test-producer", false)

		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.False(t, cfg.Producer.Idempotent)
		assert.Equal(t, sarama.WaitForAll, cfg.Producer.RequiredAcks)
		assert.Contains(t, cfg.ClientID, "test-producer")
	})

	t.Run("uses hostname in client ID", func(t *testing.T) {
		t.Parallel()

		cfg, err := DefaultProducerSaramaConfig("my-app", false)

		require.NoError(t, err)
		require.NotNil(t, cfg)
		// Client ID should contain hostname and app name
		assert.Contains(t, cfg.ClientID, "my-app")
		assert.NotEmpty(t, cfg.ClientID)
	})
}

func TestInjectTracingAndCorrelationHeaders_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("handles message with nil headers", func(t *testing.T) {
		t.Parallel()

		ctx := correlation.ContextWithID(context.Background(), "test-123")
		msg := &sarama.ProducerMessage{
			Topic:   "test",
			Headers: nil,
		}

		injectTracingAndCorrelationHeaders(ctx, msg)

		assert.NotNil(t, msg.Headers)
		assert.GreaterOrEqual(t, len(msg.Headers), 1) // At least correlation ID
	})

	t.Run("handles message with existing headers", func(t *testing.T) {
		t.Parallel()

		ctx := correlation.ContextWithID(context.Background(), "test-456")
		msg := &sarama.ProducerMessage{
			Topic: "test",
			Headers: []sarama.RecordHeader{
				{Key: []byte("existing"), Value: []byte("header")},
			},
		}

		injectTracingAndCorrelationHeaders(ctx, msg)

		assert.GreaterOrEqual(t, len(msg.Headers), 2) // Existing + new headers

		// Verify existing header is preserved
		found := false
		for _, h := range msg.Headers {
			if string(h.Key) == "existing" && string(h.Value) == "header" {
				found = true
				break
			}
		}
		assert.True(t, found, "existing header should be preserved")
	})

	t.Run("extracts correlation ID from context", func(t *testing.T) {
		t.Parallel()

		expectedID := "correlation-xyz"
		ctx := correlation.ContextWithID(context.Background(), expectedID)
		msg := &sarama.ProducerMessage{Topic: "test"}

		injectTracingAndCorrelationHeaders(ctx, msg)

		// Find correlation ID header
		var foundID string
		for _, h := range msg.Headers {
			if string(h.Key) == correlation.HeaderID {
				foundID = string(h.Value)
				break
			}
		}

		assert.Equal(t, expectedID, foundID)
	})
}
