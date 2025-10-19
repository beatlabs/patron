package cache

import (
	"context"
	"sync"

	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	packageName     = "http-cache"
	statusAttribute = "status"
)

var (
	validationReason = map[validationContext]string{0: "nil", ttlValidation: "expired", maxAgeValidation: "max_age", minFreshValidation: "min_fresh"}
	cacheMetrics     *httpCacheMetrics
	cacheMetricsOnce sync.Once
	errCacheMetrics  error
	statusAddAttr    = attribute.String(statusAttribute, "add")
	statusHitAttr    = attribute.String(statusAttribute, "hit")
	statusMissAttr   = attribute.String(statusAttribute, "miss")
	statusErrAttr    = attribute.String(statusAttribute, "err")
	statusEvictAttr  = attribute.String(statusAttribute, "evict")
)

type httpCacheMetrics struct {
	cacheExpirationHistogram metric.Int64Histogram
	cacheStatusCounter       metric.Int64Counter
}

func setupMetricsOnce() error {
	cacheMetricsOnce.Do(func() {
		var expirationHistogram metric.Int64Histogram
		var statusCounter metric.Int64Counter

		expirationHistogram, errCacheMetrics = patronmetric.Int64Histogram(packageName, "http.cache.expiration", "HTTP cache expiration.", "s")
		if errCacheMetrics != nil {
			return
		}

		statusCounter, errCacheMetrics = patronmetric.Int64Counter(packageName, "http.cache.status", "HTTP cache status.", "1")
		if errCacheMetrics != nil {
			return
		}

		cacheMetrics = &httpCacheMetrics{
			cacheExpirationHistogram: expirationHistogram,
			cacheStatusCounter:       statusCounter,
		}
	})
	return errCacheMetrics
}

func observeCacheAdd(path string) {
	_ = setupMetricsOnce()
	if cacheMetrics == nil {
		return
	}
	cacheMetrics.cacheStatusCounter.Add(context.Background(), 1, metric.WithAttributes(routeAttr(path), statusAddAttr))
}

func observeCacheMiss(path string) {
	_ = setupMetricsOnce()
	if cacheMetrics == nil {
		return
	}
	cacheMetrics.cacheStatusCounter.Add(context.Background(), 1, metric.WithAttributes(routeAttr(path), statusMissAttr))
}

func observeCacheHit(path string) {
	_ = setupMetricsOnce()
	if cacheMetrics == nil {
		return
	}
	cacheMetrics.cacheStatusCounter.Add(context.Background(), 1, metric.WithAttributes(routeAttr(path), statusHitAttr))
}

func observeCacheErr(path string) {
	_ = setupMetricsOnce()
	if cacheMetrics == nil {
		return
	}
	cacheMetrics.cacheStatusCounter.Add(context.Background(), 1, metric.WithAttributes(routeAttr(path), statusErrAttr))
}

func observeCacheEvict(path string, validationContext validationContext, age int64) {
	_ = setupMetricsOnce()
	if cacheMetrics == nil {
		return
	}
	cacheMetrics.cacheExpirationHistogram.Record(context.Background(), age, metric.WithAttributes(routeAttr(path)))
	cacheMetrics.cacheStatusCounter.Add(context.Background(), 1, metric.WithAttributes(routeAttr(path), statusEvictAttr,
		reasonAttr(validationReason[validationContext])))
}

func routeAttr(route string) attribute.KeyValue {
	return attribute.String("route", route)
}

func reasonAttr(reason string) attribute.KeyValue {
	return attribute.String("reason", reason)
}
