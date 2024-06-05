// Package es provides a client with included tracing capabilities.
package es

import (
	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/prometheus/client_golang/prometheus"
)

// TODO: introduce metrics.
var reqDurationMetrics *prometheus.HistogramVec

func init() {
	reqDurationMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "client",
			Subsystem: "elasticsearch",
			Name:      "request_duration_seconds",
			Help:      "Elasticsearch requests completed by the client.",
		},
		[]string{"method", "url", "status_code"},
	)
	prometheus.MustRegister(reqDurationMetrics)
}

// New creates a new elasticsearch client with tracing capabilities.
func New(cfg elasticsearch.Config, version string) (*elasticsearch.Client, error) {
	// TODO: metrics integration
	cfg.Instrumentation = elastictransport.NewOtelInstrumentation(nil, false, version)

	return elasticsearch.NewClient(cfg)
}
