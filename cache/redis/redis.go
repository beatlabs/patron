// Package redis contains the concrete implementation of a cache that supports TTL.
package redis

import (
	"context"
	"errors"
	"time"

	"github.com/beatlabs/patron/cache"
	patronredis "github.com/beatlabs/patron/client/redis"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	_                cache.TTLCache = &Cache{}
	redisAttribute                  = attribute.String("cache.type", "redis")
	useCaseAttribute attribute.KeyValue
)

// Cache encapsulates a Redis-based caching mechanism.
type Cache struct {
	rdb          *redis.Client
	cacheCounter metric.Int64Counter
}

// New creates a cache returns a new Redis client that will be used as the cache store.
func New(opt *redis.Options, useCase string) (*Cache, error) {
	redisDB, err := patronredis.New(opt)
	if err != nil {
		return nil, err
	}
	useCaseAttribute = attribute.String("cache.use_case", useCase)
	return &Cache{rdb: redisDB}, nil
}

// Get executes a lookup and returns whether a key exists in the cache along with its value.
func (c *Cache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	res, err := c.rdb.Do(ctx, "get", key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) { // cache miss
			cache.ObserveMiss(ctx, redisAttribute, useCaseAttribute)
			return nil, false, nil
		}
		return nil, false, err
	}
	cache.ObserveHit(ctx, redisAttribute, useCaseAttribute)
	return res, true, nil
}

// Set registers a key-value pair to the cache.
func (c *Cache) Set(ctx context.Context, key string, value interface{}) error {
	return c.rdb.Do(ctx, "set", key, value).Err()
}

// Purge evicts all keys present in the cache.
func (c *Cache) Purge(ctx context.Context) error {
	return c.rdb.FlushAll(ctx).Err()
}

// Remove evicts a specific key from the cache.
func (c *Cache) Remove(ctx context.Context, key string) error {
	return c.rdb.Do(ctx, "del", key).Err()
}

// SetTTL registers a key-value pair to the cache, specifying an expiry time.
func (c *Cache) SetTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.rdb.Do(ctx, "set", key, value, "px", int(ttl.Milliseconds())).Err()
}
