package cache

import "github.com/prometheus/client_golang/prometheus"

var validationReason = map[validationContext]string{0: "nil", ttlValidation: "expired", maxAgeValidation: "max_age", minFreshValidation: "min_fresh"}

type metrics interface {
	add(path string)
	miss(path string)
	hit(path string)
	err(path string)
	evict(path string, context validationContext, age int64)
}

// prometheusMetrics is the prometheus implementation for exposing cache metrics.
type prometheusMetrics struct {
	ageHistogram *prometheus.HistogramVec
	operations   *prometheus.CounterVec
}

func (m *prometheusMetrics) add(path string) {
	m.operations.WithLabelValues(path, "add", "").Inc()
}

func (m *prometheusMetrics) miss(path string) {
	m.operations.WithLabelValues(path, "miss", "").Inc()
}

func (m *prometheusMetrics) hit(path string) {
	m.operations.WithLabelValues(path, "hit", "").Inc()
}

func (m *prometheusMetrics) err(path string) {
	m.operations.WithLabelValues(path, "Err", "").Inc()
}

func (m *prometheusMetrics) evict(path string, context validationContext, age int64) {
	m.ageHistogram.WithLabelValues(path).Observe(float64(age))
	m.operations.WithLabelValues(path, "evict", validationReason[context]).Inc()
}

// newPrometheusMetrics constructs a new prometheus metrics implementation instance.
func newPrometheusMetrics() *prometheusMetrics {
	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "http_cache",
		Subsystem: "handler",
		Name:      "expiration",
		Help:      "Expiry age for evicted objects.",
		Buckets:   []float64{1, 10, 30, 60, 60 * 5, 60 * 10, 60 * 30, 60 * 60},
	}, []string{"route"})

	operations := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "http_cache",
		Subsystem: "handler",
		Name:      "operations",
		Help:      "Number of cache operations.",
	}, []string{"route", "operation", "reason"})

	m := &prometheusMetrics{
		ageHistogram: histogram,
		operations:   operations,
	}

	prometheus.MustRegister(m.ageHistogram, m.operations)

	return m
}
