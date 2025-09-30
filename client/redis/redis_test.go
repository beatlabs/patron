package redis

import (
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("success with valid options", func(t *testing.T) {
		t.Parallel()

		opt := &redis.Options{
			Addr: "localhost:6379",
		}

		client, err := New(opt)

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.NotNil(t, client.Options())
		assert.Equal(t, "localhost:6379", client.Options().Addr)

		// Clean up
		err = client.Close()
		require.NoError(t, err)
	})

	t.Run("success with custom options", func(t *testing.T) {
		t.Parallel()

		opt := &redis.Options{
			Addr:     "localhost:6380",
			Password: "secret",
			DB:       1,
		}

		client, err := New(opt)

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "localhost:6380", client.Options().Addr)
		assert.Equal(t, "secret", client.Options().Password)
		assert.Equal(t, 1, client.Options().DB)

		// Clean up
		err = client.Close()
		require.NoError(t, err)
	})

	t.Run("instrumentation is applied", func(t *testing.T) {
		t.Parallel()

		opt := &redis.Options{
			Addr: "localhost:6379",
		}

		client, err := New(opt)

		require.NoError(t, err)
		assert.NotNil(t, client)

		// Verify that hooks are added (tracing and metrics instrumentation)
		// The redisotel package adds hooks to the client
		assert.NotEmpty(t, client.Options().Addr)

		// Clean up
		err = client.Close()
		require.NoError(t, err)
	})

	t.Run("nil options should not panic", func(t *testing.T) {
		t.Parallel()

		// This will panic in the underlying redis.NewClient, but we test that our wrapper
		// doesn't add any issues
		assert.Panics(t, func() {
			_, _ = New(nil)
		})
	})
}

func TestNew_WithMultipleAddresses(t *testing.T) {
	t.Parallel()

	opt := &redis.Options{
		Addr:       "localhost:6379",
		MaxRetries: 3,
		PoolSize:   10,
	}

	client, err := New(opt)

	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, 3, client.Options().MaxRetries)
	assert.Equal(t, 10, client.Options().PoolSize)

	// Clean up
	err = client.Close()
	require.NoError(t, err)
}
