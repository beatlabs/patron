//go:build integration
// +build integration

package redis

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	cacheredis "github.com/beatlabs/patron/cache/redis"
	"github.com/beatlabs/patron/client/redis"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var runtime *redisRuntime

func TestMain(m *testing.M) {
	var err error
	runtime, err = create(60 * time.Second)
	if err != nil {
		fmt.Printf("could not create mysql runtime: %v\n", err)
		os.Exit(1)
	}
	defer func() {
	}()
	exitCode := m.Run()

	ee := runtime.Teardown()
	if len(ee) > 0 {
		for _, err = range ee {
			fmt.Printf("could not tear down containers: %v\n", err)
		}
	}
	os.Exit(exitCode)
}

func TestCache(t *testing.T) {
	dsn, err := runtime.DSN()
	require.NoError(t, err)

	cache, err := cacheredis.New(context.Background(), cacheredis.Options{
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

func TestClient(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	defer mtr.Reset()
	dsn, err := runtime.DSN()
	require.NoError(t, err)

	cl := redis.New(redis.Options{
		Addr:     dsn,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	cmd := cl.Set(context.Background(), "key", "value", 0)
	res, err := cmd.Result()
	assert.NoError(t, err)
	assert.Equal(t, res, "OK")
	assert.Len(t, mtr.FinishedSpans(), 1)
	assert.Equal(t, mtr.FinishedSpans()[0].Tags()["component"], "redis")
	assert.Equal(t, mtr.FinishedSpans()[0].Tags()["error"], false)
	assert.Regexp(t, `:\d+`, mtr.FinishedSpans()[0].Tags()["db.instance"])
	assert.Equal(t, mtr.FinishedSpans()[0].Tags()["db.statement"], "set key value")
	assert.Equal(t, mtr.FinishedSpans()[0].Tags()["db.type"], "kv")
}
