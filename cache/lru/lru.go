// Package lru implements a LRU based cache.
package lru

import (
	"context"

	"github.com/beatlabs/patron/cache"
	lru "github.com/hashicorp/golang-lru"
	"go.opentelemetry.io/otel/attribute"
)

var (
	_                cache.Cache = &Cache{}
	lruAttribute                 = attribute.String("cache.type", "lru")
	useCaseAttribute attribute.KeyValue
)

// Cache encapsulates a thread-safe fixed size LRU cache.
type Cache struct {
	cache *lru.Cache
}

// New returns a new LRU cache that can hold 'size' number of keys at a time.
func New(size int, useCase string) (*Cache, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	useCaseAttribute = attribute.String("cache.use_case", useCase)
	return &Cache{cache: cache}, nil
}

// Get executes a lookup and returns whether a key exists in the cache along with its value.
func (c *Cache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	value, ok := c.cache.Get(key)
	if !ok {
		cache.ObserveMiss(ctx, lruAttribute, useCaseAttribute)
		return nil, false, nil
	}
	cache.ObserveHit(ctx, lruAttribute, useCaseAttribute)
	return value, true, nil
}

// Purge evicts all keys present in the cache.
func (c *Cache) Purge(_ context.Context) error {
	c.cache.Purge()
	return nil
}

// Remove evicts a specific key from the cache.
func (c *Cache) Remove(_ context.Context, key string) error {
	c.cache.Remove(key)
	return nil
}

// Set registers a key-value pair to the cache.
func (c *Cache) Set(_ context.Context, key string, value interface{}) error {
	c.cache.Add(key, value)
	return nil
}
