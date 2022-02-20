package v2

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// MetricsPath of the component.
	MetricsPath = "/metrics"
)

func MetricRoute() *Route {
	route, _ := NewRoute(http.MethodGet, MetricsPath, promhttp.Handler().ServeHTTP)
	return route
}

func ProfilingRoutes() []*Route {
	var routes []*Route

	routeFunc := func(path string, handler http.HandlerFunc) *Route {
		route, _ := NewRoute(http.MethodGet, path, handler)
		return route
	}

	routes = append(routes, routeFunc("/debug/pprof/", pprof.Index))
	routes = append(routes, routeFunc("/debug/pprof/cmdline/", pprof.Cmdline))
	routes = append(routes, routeFunc("/debug/pprof/profile/", pprof.Profile))
	routes = append(routes, routeFunc("/debug/pprof/symbol/", pprof.Symbol))
	routes = append(routes, routeFunc("/debug/pprof/trace/", pprof.Trace))
	routes = append(routes, routeFunc("/debug/pprof/allocs/", pprof.Handler("allocs").ServeHTTP))
	routes = append(routes, routeFunc("/debug/pprof/heap/", pprof.Handler("heap").ServeHTTP))
	routes = append(routes, routeFunc("/debug/pprof/goroutine/", pprof.Handler("goroutine").ServeHTTP))
	routes = append(routes, routeFunc("/debug/pprof/block/", pprof.Handler("block").ServeHTTP))
	routes = append(routes, routeFunc("/debug/pprof/threadcreate/", pprof.Handler("threadcreate").ServeHTTP))
	routes = append(routes, routeFunc("/debug/pprof/mutex/", pprof.Handler("mutex").ServeHTTP))
	routes = append(routes, routeFunc("/debug/vars/", expVars))

	return routes
}

// Replicated from expvar.go as not public.
func expVars(w http.ResponseWriter, r *http.Request) {
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
