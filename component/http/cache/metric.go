package cache

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

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
	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "http_cache",
		Subsystem: "handler",
		Name:      "expiration",
		Help:      "Expiry age for evicted objects.",
		Buckets:   []float64{1, 10, 30, 60, 60 * 5, 60 * 10, 60 * 30, 60 * 60},
	}, []string{"route"})
	prometheus.MustRegister(histogram)

	operations := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "http_cache",
		Subsystem: "handler",
		Name:      "operations",
		Help:      "Number of cache operations.",
	}, []string{"route", "operation", "reason"})
	prometheus.MustRegister(operations)

	var err error
	cacheExpirationHistogram, err = otel.Meter("http_cache").Int64Histogram("http.cache.expiration",
		metric.WithDescription("HTTP cache expiration."),
		metric.WithUnit("s"))
	if err != nil {
		panic(err)
	}

	cacheStatusCounter, err = otel.Meter("http_cache").Int64Counter("http.cache.status",
		metric.WithDescription("HTTP cache status."),
		metric.WithUnit("1"))
	if err != nil {
		panic(err)
	}
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
