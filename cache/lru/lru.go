// Package lru implements a LRU based cache.
package lru

import (
	"context"

	"github.com/beatlabs/patron/cache"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.opentelemetry.io/otel/attribute"
)

const cacheTypeAttrbute = "cache.type"

var (
	lruAttribute      = attribute.String(cacheTypeAttrbute, "lru")
	lruEvictAttribute = attribute.String(cacheTypeAttrbute, "lru-evict")
)

// Cache encapsulates a thread-safe fixed size LRU cache.
type Cache[k comparable, v any] struct {
	cache            *lru.Cache[k, v]
	useCaseAttribute attribute.KeyValue
	typeAttribute    attribute.KeyValue
}

// New returns a new LRU cache that can hold 'size' number of keys at a time.
func New[k comparable, v any](size int, useCase string) (*Cache[k, v], error) {
	cache.SetupMetricsOnce()
	chc, err := lru.New[k, v](size)
	if err != nil {
		return nil, err
	}

	return newFunction(chc, lruAttribute, cache.UseCaseAttribute(useCase)), nil
}

// NewWithEvict returns a new LRU cache that can hold 'size' number of keys at a time.
func NewWithEvict[k comparable, v any](size int, useCase string, onEvict func(k, v)) (*Cache[k, v], error) {
	cache.SetupMetricsOnce()

	chc, err := lru.NewWithEvict[k, v](size, func(key k, value v) {
		onEvict(key, value)
		cache.ObserveEviction(context.Background(), lruEvictAttribute, cache.UseCaseAttribute(useCase))
	})
	if err != nil {
		return nil, err
	}

	return newFunction(chc, lruEvictAttribute, cache.UseCaseAttribute(useCase)), nil
}

func newFunction[k comparable, v any](chc *lru.Cache[k, v], typeAttr attribute.KeyValue,
	useCaseAttr attribute.KeyValue,
) *Cache[k, v] {
	return &Cache[k, v]{
		cache:            chc,
		typeAttribute:    typeAttr,
		useCaseAttribute: useCaseAttr,
	}
}

// Get executes a lookup and returns whether a key exists in the cache along with its value.
func (c *Cache[k, v]) Get(ctx context.Context, key k) (any, bool, error) {
	value, ok := c.cache.Get(key)
	if !ok {
		cache.ObserveMiss(ctx, lruAttribute, c.useCaseAttribute, c.typeAttribute)
		return nil, false, nil
	}
	cache.ObserveHit(ctx, lruAttribute, c.useCaseAttribute, c.typeAttribute)
	return value, true, nil
}

// Purge evicts all keys present in the cache.
func (c *Cache[k, v]) Purge(_ context.Context) error {
	c.cache.Purge()
	return nil
}

// Remove evicts a specific key from the cache.
func (c *Cache[k, v]) Remove(_ context.Context, key k) error {
	c.cache.Remove(key)
	return nil
}

// Set registers a key-value pair to the cache.
func (c *Cache[k, v]) Set(_ context.Context, key k, value v) error {
	c.cache.Add(key, value)
	return nil
}
