// Package cache provides abstractions to allow the creation of concrete implementations.
package cache

import (
	"context"
	"time"

	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	cashHitAttribute  = attribute.String("cache.status", "hit")
	cashMissAttribute = attribute.String("cache.status", "miss")
	cacheCounter      metric.Int64Counter
)

// TODO: move metric to each client implementation.

func init() {
	var err error
	cacheCounter, err = patronmetric.Meter().Int64Counter("cache.counter",
		metric.WithDescription("Number of cache calls."),
		metric.WithUnit("1"),
	)
	if err != nil {
		panic(err)
	}
}

// ObserveHit increments the cache hit counter.
func ObserveHit(ctx context.Context, attrs ...attribute.KeyValue) {
	attrs = append(attrs, cashHitAttribute)
	cacheCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// ObserveMiss increments the cache miss counter.
func ObserveMiss(ctx context.Context, attrs ...attribute.KeyValue) {
	attrs = append(attrs, cashMissAttribute)
	cacheCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Cache interface that defines the methods required.
type Cache interface {
	// Get a value based on a specific key. The call returns whether the  key exists or not.
	Get(ctx context.Context, key string) (interface{}, bool, error)
	// Purge the cache.
	Purge(ctx context.Context) error
	// Remove the key from the cache.
	Remove(ctx context.Context, key string) error
	// Set the value for the specified key.
	Set(ctx context.Context, key string, value interface{}) error
}

// TTLCache interface adds support for expiring key value pairs.
type TTLCache interface {
	Cache
	// SetTTL sets the value of a specified key with a time to live.
	SetTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}
