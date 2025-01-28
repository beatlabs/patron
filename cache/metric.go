package cache

import (
	"context"
	"sync"

	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	packageName          = "cache"
	cacheStatusAttribute = "cache.status"
)

var (
	cacheHitAttribute   = attribute.String(cacheStatusAttribute, "hit")
	cacheMissAttribute  = attribute.String(cacheStatusAttribute, "miss")
	cacheEvictAttribute = attribute.String(cacheStatusAttribute, "evict")
	cacheCounter        metric.Int64Counter
	cacheOnce           sync.Once
)

// SetupMetricsOnce initializes the cache counter.
func SetupMetricsOnce() {
	cacheOnce.Do(func() {
		cacheCounter = patronmetric.Int64Counter(packageName, "cache.counter", "Number of cache calls.", "1")
	})
}

// UseCaseAttribute returns an attribute.KeyValue with the use case.
func UseCaseAttribute(useCase string) attribute.KeyValue {
	return attribute.String("cache.use_case", useCase)
}

// ObserveHit increments the cache hit counter.
func ObserveHit(ctx context.Context, attrs ...attribute.KeyValue) {
	attrs = append(attrs, cacheHitAttribute)
	cacheCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// ObserveMiss increments the cache miss counter.
func ObserveMiss(ctx context.Context, attrs ...attribute.KeyValue) {
	attrs = append(attrs, cacheMissAttribute)
	cacheCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func ObserveEviction(ctx context.Context, attrs ...attribute.KeyValue) {
	attrs = append(attrs, cacheEvictAttribute)
	cacheCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}
