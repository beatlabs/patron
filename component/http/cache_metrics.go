package http

import "github.com/prometheus/client_golang/prometheus"

var validationReason = map[validationContext]string{0: "nil", ttlValidation: "expired", maxAgeValidation: "max_age", minFreshValidation: "min_fresh", maxStaleValidation: "max_stale"}

type cacheMetrics interface {
	add(key string)
	miss(key string)
	hit(key string)
	evict(key string, context validationContext, age int64)
	reset() bool
}

type prometheusMetrics struct {
	path         string
	expiry       *prometheus.GaugeVec
	ageHistogram *prometheus.HistogramVec
	misses       *prometheus.CounterVec
	additions    *prometheus.CounterVec
	hits         *prometheus.CounterVec
	evictions    *prometheus.CounterVec
}

func (m *prometheusMetrics) add(key string) {
	m.additions.WithLabelValues(m.path).Inc()
}

func (m *prometheusMetrics) miss(key string) {
	m.misses.WithLabelValues(m.path).Inc()
}

func (m *prometheusMetrics) hit(key string) {
	m.hits.WithLabelValues(m.path).Inc()
}

func (m *prometheusMetrics) evict(key string, context validationContext, age int64) {
	m.ageHistogram.WithLabelValues(m.path).Observe(float64(age))
	m.evictions.WithLabelValues(m.path, validationReason[context]).Inc()
}

func (m *prometheusMetrics) reset() bool {
	exp := prometheus.DefaultRegisterer.Unregister(m.expiry)
	hist := prometheus.DefaultRegisterer.Unregister(m.ageHistogram)
	miss := prometheus.DefaultRegisterer.Unregister(m.misses)
	hits := prometheus.DefaultRegisterer.Unregister(m.hits)
	evict := prometheus.DefaultRegisterer.Unregister(m.evictions)
	add := prometheus.DefaultRegisterer.Unregister(m.additions)
	return exp && hist && miss && evict && hits && add
}

func NewPrometheusMetrics(path string, expiry int64) *prometheusMetrics {

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

	evictions := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "http_cache",
		Subsystem: "handler",
		Name:      "evicts",
		Help:      "Number of cache evictions.",
	}, []string{"route", "reason"})

	expiration := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "http_cache",
		Subsystem: "handler",
		Name:      "time_to_live",
		Help:      "Expiration parameter of the http cache.",
	}, []string{"route"})

	prometheus.MustRegister(histogram, additions, misses, hits, evictions, expiration)

	expiration.WithLabelValues(path).Set(float64(expiry))

	return &prometheusMetrics{
		path:         path,
		expiry:       expiration,
		ageHistogram: histogram,
		additions:    additions,
		misses:       misses,
		hits:         hits,
		evictions:    evictions,
	}

}

type voidMetrics struct {
}

func NewVoidMetrics() *voidMetrics {
	return &voidMetrics{}
}

func (v *voidMetrics) add(key string) {
	// do nothing
}

func (v *voidMetrics) miss(key string) {
	// do nothing
}

func (v *voidMetrics) hit(key string) {
	// do nothing
}

func (v *voidMetrics) evict(key string, context validationContext, age int64) {
	// do nothing
}

func (v *voidMetrics) reset() bool {
	// do nothing
	return true
}
