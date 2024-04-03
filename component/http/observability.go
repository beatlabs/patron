package http

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// MetricsPath of the component.
	MetricsPath = "GET /metrics"
)

// MetricRoute creation.
func MetricRoute() *Route {
	return &Route{
		path:    MetricsPath,
		handler: promhttp.Handler().ServeHTTP,
	}
}

func ProfilingRoutes(enableExpVar bool) []*Route {
	var routes []*Route

	routeFunc := func(path string, handler http.HandlerFunc) *Route {
		route, _ := NewRoute(path, handler)
		return route
	}

	routes = append(routes, routeFunc("GET /debug/pprof/", pprof.Index))
	routes = append(routes, routeFunc("GET /debug/pprof/cmdline/", pprof.Cmdline))
	routes = append(routes, routeFunc("GET /debug/pprof/profile/", pprof.Profile))
	routes = append(routes, routeFunc("GET /debug/pprof/symbol/", pprof.Symbol))
	routes = append(routes, routeFunc("GET /debug/pprof/trace/", pprof.Trace))
	routes = append(routes, routeFunc("GET /debug/pprof/allocs/", pprof.Handler("allocs").ServeHTTP))
	routes = append(routes, routeFunc("GET /debug/pprof/heap/", pprof.Handler("heap").ServeHTTP))
	routes = append(routes, routeFunc("GET /debug/pprof/goroutine/", pprof.Handler("goroutine").ServeHTTP))
	routes = append(routes, routeFunc("GET /debug/pprof/block/", pprof.Handler("block").ServeHTTP))
	routes = append(routes, routeFunc("GET /debug/pprof/threadcreate/", pprof.Handler("threadcreate").ServeHTTP))
	routes = append(routes, routeFunc("GET /debug/pprof/mutex/", pprof.Handler("mutex").ServeHTTP))
	if enableExpVar {
		routes = append(routes, routeFunc("GET /debug/vars/", expVars))
	}

	return routes
}

// Replicated from expvar.go as not public.
func expVars(w http.ResponseWriter, _ *http.Request) {
	first := true
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, "{\n")
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			_, _ = fmt.Fprintf(w, ",\n")
		}
		first = false
		_, _ = fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	_, _ = fmt.Fprintf(w, "\n}\n")
}
