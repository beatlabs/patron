//go:build integration
// +build integration

package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dsn = "localhost:6379"
)

func TestCache(t *testing.T) {
	cache, err := New(context.Background(), Options{
		Addr:     dsn,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	require.NoError(t, err)

	key1 := "key1"
	val1 := "value1"
	key2 := "key2"
	val2 := "value2"
	key3 := "key3"
	val3 := "value3"

	t.Run("set", func(t *testing.T) {
		assert.NoError(t, cache.Set(key1, val1))
	})

	t.Run("get", func(t *testing.T) {
		got, exists, err := cache.Get(key1)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, val1, got)
	})

	t.Run("delete", func(t *testing.T) {
		assert.NoError(t, cache.Remove(key1))
		_, exists, err := cache.Get(key1)
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("ttl", func(t *testing.T) {
		assert.NoError(t, cache.SetTTL(key1, val1, 2*time.Millisecond))
		got, exists, err := cache.Get(key1)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, val1, got)
		time.Sleep(10 * time.Millisecond)
		_, exists, err = cache.Get(key1)
		assert.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("purge", func(t *testing.T) {
		assert.NoError(t, cache.Set(key1, val1))
		assert.NoError(t, cache.Set(key2, val2))
		assert.NoError(t, cache.Set(key3, val3))

		assert.NoError(t, cache.Purge())
		_, exists, err := cache.Get(key1)
		assert.NoError(t, err)
		assert.False(t, exists)
		_, exists, err = cache.Get(key2)
		assert.NoError(t, err)
		assert.False(t, exists)
		_, exists, err = cache.Get(key3)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}
