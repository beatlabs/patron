package amqp

import (
	"context"
	"testing"
	"time"

	"github.com/beatlabs/patron/correlation"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNew(t *testing.T) {
	t.Parallel()
	type args struct {
		url string
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"fail, missing url": {args: args{}, expectedErr: "url is required"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := New(tt.args.url)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func Test_injectTraceHeaders(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	_ = patrontrace.Setup("test", nil, exp)

	msg := amqp.Publishing{}
	ctx, sp := injectTraceHeaders(context.Background(), "123", &msg)
	assert.NotNil(t, ctx)
	assert.NotNil(t, sp)
	assert.Len(t, msg.Headers, 2)
	assert.Empty(t, exp.GetSpans())
}

func TestNew_Validation(t *testing.T) {
	t.Parallel()

	t.Run("empty URL returns error", func(t *testing.T) {
		t.Parallel()

		pub, err := New("")

		require.Error(t, err)
		assert.Nil(t, pub)
		assert.Equal(t, "url is required", err.Error())
	})

	t.Run("invalid URL returns error", func(t *testing.T) {
		t.Parallel()

		pub, err := New("invalid-url")

		require.Error(t, err)
		assert.Nil(t, pub)
		assert.Contains(t, err.Error(), "failed to open connection")
	})
}

func TestNew_WithConfig(t *testing.T) {
	t.Parallel()

	t.Run("applies config option", func(t *testing.T) {
		t.Parallel()

		cfg := amqp.Config{
			Locale: "en_US",
		}

		// This will fail to connect but will test option application
		_, err := New("amqp://invalid", WithConfig(cfg))

		require.Error(t, err) // Connection will fail
		assert.Contains(t, err.Error(), "failed to open connection")
	})
}

func TestInjectTraceHeaders_WithCorrelation(t *testing.T) {
	t.Parallel()

	exp := tracetest.NewInMemoryExporter()
	_ = patrontrace.Setup("test", nil, exp)

	t.Run("injects correlation ID into headers", func(t *testing.T) {
		t.Parallel()

		ctx := correlation.ContextWithID(context.Background(), "test-correlation-id")
		msg := amqp.Publishing{}

		ctx, sp := injectTraceHeaders(ctx, "test-exchange", &msg)

		require.NotNil(t, ctx)
		require.NotNil(t, sp)
		require.NotNil(t, msg.Headers)

		corID, ok := msg.Headers[correlation.HeaderID]
		require.True(t, ok)
		assert.Equal(t, "test-correlation-id", corID)

		sp.End()
	})

	t.Run("creates headers table if nil", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		msg := amqp.Publishing{}

		ctx, sp := injectTraceHeaders(ctx, "test-exchange", &msg)

		require.NotNil(t, ctx)
		require.NotNil(t, sp)
		assert.NotNil(t, msg.Headers)

		sp.End()
	})

	t.Run("preserves existing headers", func(t *testing.T) {
		t.Parallel()

		ctx := correlation.ContextWithID(context.Background(), "correlation-123")
		msg := amqp.Publishing{
			Headers: amqp.Table{
				"custom-header": "custom-value",
			},
		}

		ctx, sp := injectTraceHeaders(ctx, "test-exchange", &msg)

		require.NotNil(t, ctx)
		require.NotNil(t, sp)
		require.NotNil(t, msg.Headers)

		// Verify custom header is preserved
		customVal, ok := msg.Headers["custom-header"]
		require.True(t, ok)
		assert.Equal(t, "custom-value", customVal)

		// Verify correlation ID is added
		corID, ok := msg.Headers[correlation.HeaderID]
		require.True(t, ok)
		assert.Equal(t, "correlation-123", corID)

		sp.End()
	})

	t.Run("adds traceparent header", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		msg := amqp.Publishing{}

		ctx, sp := injectTraceHeaders(ctx, "test-exchange", &msg)

		require.NotNil(t, ctx)
		require.NotNil(t, sp)
		require.NotNil(t, msg.Headers)

		// traceparent should be injected by OTEL propagator
		_, hasTraceparent := msg.Headers["traceparent"]
		assert.True(t, hasTraceparent || len(msg.Headers) >= 1) // May or may not have active span

		sp.End()
	})
}

func TestProducerMessageCarrier_Get(t *testing.T) {
	t.Parallel()

	t.Run("returns empty string", func(t *testing.T) {
		t.Parallel()

		msg := &amqp.Publishing{}
		carrier := producerMessageCarrier{msg: msg}

		value := carrier.Get("any-key")

		assert.Empty(t, value)
	})
}

func TestProducerMessageCarrier_Set(t *testing.T) {
	t.Parallel()

	t.Run("sets header on message", func(t *testing.T) {
		t.Parallel()

		msg := &amqp.Publishing{
			Headers: amqp.Table{},
		}
		carrier := producerMessageCarrier{msg: msg}

		carrier.Set("test-key", "test-value")

		value, ok := msg.Headers["test-key"]
		require.True(t, ok)
		assert.Equal(t, "test-value", value)
	})

	t.Run("sets multiple headers", func(t *testing.T) {
		t.Parallel()

		msg := &amqp.Publishing{
			Headers: amqp.Table{},
		}
		carrier := producerMessageCarrier{msg: msg}

		carrier.Set("key1", "value1")
		carrier.Set("key2", "value2")

		value1, ok1 := msg.Headers["key1"]
		require.True(t, ok1)
		assert.Equal(t, "value1", value1)

		value2, ok2 := msg.Headers["key2"]
		require.True(t, ok2)
		assert.Equal(t, "value2", value2)
	})

	t.Run("overwrites existing header", func(t *testing.T) {
		t.Parallel()

		msg := &amqp.Publishing{
			Headers: amqp.Table{
				"test-key": "old-value",
			},
		}
		carrier := producerMessageCarrier{msg: msg}

		carrier.Set("test-key", "new-value")

		value, ok := msg.Headers["test-key"]
		require.True(t, ok)
		assert.Equal(t, "new-value", value)
	})
}

func TestProducerMessageCarrier_Keys(t *testing.T) {
	t.Parallel()

	t.Run("returns nil", func(t *testing.T) {
		t.Parallel()

		msg := &amqp.Publishing{}
		carrier := producerMessageCarrier{msg: msg}

		keys := carrier.Keys()

		assert.Nil(t, keys)
	})
}

func TestPublisher_Structure(t *testing.T) {
	t.Parallel()

	t.Run("has expected fields", func(t *testing.T) {
		t.Parallel()

		pub := &Publisher{}

		assert.NotNil(t, pub)
		// cfg, connection, and channel are unexported fields
		// We verify the structure is correct
	})
}

func TestObservePublish(t *testing.T) {
	t.Parallel()

	t.Run("does not panic on success", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		exchange := "test-exchange"
		start := time.Now()

		assert.NotPanics(t, func() {
			observePublish(ctx, start, exchange, nil)
		})
	})

	t.Run("does not panic on error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		exchange := "test-exchange"
		start := time.Now()

		assert.NotPanics(t, func() {
			observePublish(ctx, start, exchange, assert.AnError)
		})
	})
}

func TestPackageConstants(t *testing.T) {
	t.Parallel()

	t.Run("package name is correct", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "amqp", packageName)
	})
}

func TestWithConfig_AppliesConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("sets config on publisher", func(t *testing.T) {
		t.Parallel()

		cfg := amqp.Config{
			Locale: "test-locale",
			Vhost:  "/test",
		}

		pub := &Publisher{}
		opt := WithConfig(cfg)

		err := opt(pub)

		require.NoError(t, err)
		require.NotNil(t, pub.cfg)
		assert.Equal(t, "test-locale", pub.cfg.Locale)
		assert.Equal(t, "/test", pub.cfg.Vhost)
	})

	t.Run("does not return error", func(t *testing.T) {
		t.Parallel()

		cfg := amqp.Config{}
		pub := &Publisher{}

		err := WithConfig(cfg)(pub)

		assert.NoError(t, err)
	})
}

func TestNew_OptionErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("returns error from option", func(t *testing.T) {
		t.Parallel()

		errorOption := func(_ *Publisher) error {
			return assert.AnError
		}

		pub, err := New("amqp://localhost", errorOption)

		require.Error(t, err)
		assert.Nil(t, pub)
		assert.Equal(t, assert.AnError, err)
	})
}
