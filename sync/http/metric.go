package http

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func metricRoute() *RouteBuilder {
	return NewRawRouteBuilder("/metrics", promhttp.Handler().ServeHTTP).MethodGet()
}
