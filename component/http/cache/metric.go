package cache

import (
	"context"

	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const packageName = "http-cache"

var (
	validationReason         = map[validationContext]string{0: "nil", ttlValidation: "expired", maxAgeValidation: "max_age", minFreshValidation: "min_fresh"}
	cacheExpirationHistogram metric.Int64Histogram
	cacheStatusCounter       metric.Int64Counter
	statusAddAttr            = attribute.String("status", "add")
	statusHitAttr            = attribute.String("status", "hit")
	statusMissAttr           = attribute.String("status", "miss")
	statusErrAttr            = attribute.String("status", "err")
	statusEvictAttr          = attribute.String("status", "evict")
)

func init() {
	cacheExpirationHistogram = patronmetric.Int64Histogram(packageName, "http.cache.expiration", "HTTP cache expiration.", "s")
	cacheStatusCounter = patronmetric.Int64Counter(packageName, "http.cache.status", "HTTP cache status.", "1")
}

func observeCacheAdd(path string) {
	cacheStatusCounter.Add(context.Background(), 1, metric.WithAttributes(routeAttr(path), statusAddAttr))
}

func observeCacheMiss(path string) {
	cacheStatusCounter.Add(context.Background(), 1, metric.WithAttributes(routeAttr(path), statusMissAttr))
}

func observeCacheHit(path string) {
	cacheStatusCounter.Add(context.Background(), 1, metric.WithAttributes(routeAttr(path), statusHitAttr))
}

func observeCacheErr(path string) {
	cacheStatusCounter.Add(context.Background(), 1, metric.WithAttributes(routeAttr(path), statusErrAttr))
}

func observeCacheEvict(path string, validationContext validationContext, age int64) {
	cacheExpirationHistogram.Record(context.Background(), age, metric.WithAttributes(routeAttr(path)))
	cacheStatusCounter.Add(context.Background(), 1, metric.WithAttributes(routeAttr(path), statusEvictAttr,
		reasonAttr(validationReason[validationContext])))
}

func routeAttr(route string) attribute.KeyValue {
	return attribute.String("route", route)
}

func reasonAttr(reason string) attribute.KeyValue {
	return attribute.String("reason", reason)
}
