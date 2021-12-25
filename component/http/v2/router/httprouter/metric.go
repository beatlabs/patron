package httprouter

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	metricsPath = "/metrics"
)

func metricRoute() *Route {
	route, _ := NewRecoveryGetRoute(metricsPath, promhttp.Handler().ServeHTTP)
	return route
}
