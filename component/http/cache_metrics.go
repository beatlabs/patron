package http

import "github.com/prometheus/client_golang/prometheus"

var validationReason = map[validationContext]string{0: "nil", ttlValidation: "expired", maxAgeValidation: "max_age", minFreshValidation: "min_fresh", maxStaleValidation: "max_stale"}

// PrometheusMetrics is the prometheus implementation for exposing cache metrics
type PrometheusMetrics struct {
	ageHistogram *prometheus.HistogramVec
	misses       *prometheus.CounterVec
	additions    *prometheus.CounterVec
	hits         *prometheus.CounterVec
	errors       *prometheus.CounterVec
	evictions    *prometheus.CounterVec
}

func (m *PrometheusMetrics) add(path, key string) {
	m.additions.WithLabelValues(path).Inc()
}

func (m *PrometheusMetrics) miss(path, key string) {
	m.misses.WithLabelValues(path).Inc()
}

func (m *PrometheusMetrics) hit(path, key string) {
	m.hits.WithLabelValues(path).Inc()
}

func (m *PrometheusMetrics) err(path, key string) {
	m.errors.WithLabelValues(path).Inc()
}

func (m *PrometheusMetrics) evict(path, key string, context validationContext, age int64) {
	m.ageHistogram.WithLabelValues(path).Observe(float64(age))
	m.evictions.WithLabelValues(path, validationReason[context]).Inc()
}

func (m *PrometheusMetrics) mustRegister(registerer prometheus.Registerer) {
	registerer.MustRegister(m.ageHistogram, m.additions, m.misses, m.hits, m.evictions)
}

// NewPrometheusMetrics constructs a new prometheus metrics implementation instance
func NewPrometheusMetrics() *PrometheusMetrics {

	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "http_cache",
		Subsystem: "handler",
		Name:      "expiration",
		Help:      "Expiry age for evicted objects.",
		Buckets:   []float64{1, 10, 30, 60, 60 * 5, 60 * 10, 60 * 30, 60 * 60},
	}, []string{"route"})

	additions := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "http_cache",
		Subsystem: "handler",
		Name:      "adds",
		Help:      "Number of Added objects to the cache.",
	}, []string{"route"})

	misses := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "http_cache",
		Subsystem: "handler",
		Name:      "misses",
		Help:      "Number of cache missed.",
	}, []string{"route"})

	hits := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "http_cache",
		Subsystem: "handler",
		Name:      "hits",
		Help:      "Number of cache hits.",
	}, []string{"route"})

	errors := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "http_cache",
		Subsystem: "handler",
		Name:      "errors",
		Help:      "Number of cache errors.",
	}, []string{"route"})

	evictions := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "http_cache",
		Subsystem: "handler",
		Name:      "evicts",
		Help:      "Number of cache evictions.",
	}, []string{"route", "reason"})

	return &PrometheusMetrics{
		ageHistogram: histogram,
		additions:    additions,
		misses:       misses,
		hits:         hits,
		errors:       errors,
		evictions:    evictions,
	}

}
