package cache

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	cashHitAttribute  = attribute.String("cache.status", "hit")
	cashMissAttribute = attribute.String("cache.status", "miss")
	cacheCounter      metric.Int64Counter
	cacheOnce         sync.Once
)

// SetupMetricsOnce initializes the cache counter.
func SetupMetricsOnce() {
	cacheOnce.Do(func() {
		var err error
		cacheCounter, err = otel.Meter("cache").Int64Counter("cache.counter",
			metric.WithDescription("Number of cache calls."),
			metric.WithUnit("1"),
		)
		if err != nil {
			panic(err)
		}
	})
}

// UseCaseAttribute returns an attribute.KeyValue with the use case.
func UseCaseAttribute(useCase string) attribute.KeyValue {
	return attribute.String("cache.use_case", useCase)
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
