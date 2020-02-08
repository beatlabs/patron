package redis

import (
	"context"
	"time"

	"github.com/beatlabs/patron/client/redis"
)

// Cache encapsulates a Redis-based caching mechanism,
// driven by go-redis/redis/v7.
type Cache struct {
	rdb *redis.Client
	ctx context.Context
}

// Options exposes the options struct from go-redis package.
type Options redis.Options

// New returns a new Redis client that will be used as the cache store.
func New(ctx context.Context, opt Options) (*Cache, error) {
	redisDB := redis.New(redis.Options(opt))
	return &Cache{rdb: redisDB, ctx: ctx}, nil
}

// Get executes a lookup and returns whether a key exists in the cache along with its value.
func (c *Cache) Get(key string) (interface{}, bool, error) {
	res, err := c.rdb.Do(c.ctx, "get", key).Result()
	if err == redis.Nil || err != nil {
		return nil, false, err
	}
	return res, true, nil
}

// Set registers a key-value pair to the cache.
func (c *Cache) Set(key string, value interface{}) error {
	return c.rdb.Do(c.ctx, "set", key, value).Err()
}

// Purge evicts all keys present in the cache.
func (c *Cache) Purge() error {
	return c.rdb.Do(c.ctx, "flushdb").Err()
}

// Remove evicts a specific key from the cache.
func (c *Cache) Remove(key string) error {
	return c.rdb.Do(c.ctx, "del", key).Err()
}

// SetTTL registers a key-value pair to the cache, specifying an expiry time.
func (c *Cache) SetTTL(key string, value interface{}, ttl time.Duration) error {
	return c.rdb.Do(c.ctx, "set", key, value, "px", int(ttl.Milliseconds())).Err()
}
