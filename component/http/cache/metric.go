package cache

import (
	"context"

	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	packageName     = "http-cache"
	statusAttribute = "status"
)

var (
	validationReason         = map[validationContext]string{0: "nil", ttlValidation: "expired", maxAgeValidation: "max_age", minFreshValidation: "min_fresh"}
	cacheExpirationHistogram metric.Int64Histogram
	cacheStatusCounter       metric.Int64Counter
	statusAddAttr            = attribute.String(statusAttribute, "add")
	statusHitAttr            = attribute.String(statusAttribute, "hit")
	statusMissAttr           = attribute.String(statusAttribute, "miss")
	statusErrAttr            = attribute.String(statusAttribute, "err")
	statusEvictAttr          = attribute.String(statusAttribute, "evict")
)

func init() {
	cacheExpirationHistogram = patronmetric.Int64Histogram(packageName, "http.cache.expiration", "HTTP cache expiration.", "s")
	cacheStatusCounter = patronmetric.Int64Counter(packageName, "http.cache.status", "HTTP cache status.", "1")
}

func observeCacheAdd(ctx context.Context, path string) {
	cacheStatusCounter.Add(ctx, 1, metric.WithAttributes(routeAttr(path), statusAddAttr))
}

func observeCacheMiss(ctx context.Context, path string) {
	cacheStatusCounter.Add(ctx, 1, metric.WithAttributes(routeAttr(path), statusMissAttr))
}

func observeCacheHit(ctx context.Context, path string) {
	cacheStatusCounter.Add(ctx, 1, metric.WithAttributes(routeAttr(path), statusHitAttr))
}

func observeCacheErr(ctx context.Context, path string) {
	cacheStatusCounter.Add(ctx, 1, metric.WithAttributes(routeAttr(path), statusErrAttr))
}

func observeCacheEvict(ctx context.Context, path string, validationContext validationContext, age int64) {
	cacheExpirationHistogram.Record(ctx, age, metric.WithAttributes(routeAttr(path)))
	cacheStatusCounter.Add(ctx, 1, metric.WithAttributes(routeAttr(path), statusEvictAttr,
		reasonAttr(validationReason[validationContext])))
}

func routeAttr(route string) attribute.KeyValue {
	return attribute.String("route", route)
}

func reasonAttr(reason string) attribute.KeyValue {
	return attribute.String("reason", reason)
}
