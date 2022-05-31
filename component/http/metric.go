package http

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsPath definition.
const MetricsPath = "/metrics"

func metricRoute() *RouteBuilder {
	return NewRawRouteBuilder(MetricsPath, promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{EnableOpenMetrics: true}).ServeHTTP).MethodGet()
}
