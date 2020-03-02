package http

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func metricRoute() *RouteBuilder {
	return NewRawRouteBuilder(MethodGet, "/metrics", promhttp.Handler().ServeHTTP)
}
