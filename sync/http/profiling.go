package http

import (
	"net/http"
	"net/http/pprof"
)

func profilingRoutes() []*RouteBuilder {
	return []*RouteBuilder{
		NewRawRouteBuilder(MethodGet, "/debug/pprof/", profIndex),
		NewRawRouteBuilder(MethodGet, "/debug/pprof/allocs/", pprofAllocsIndex),
		NewRawRouteBuilder(MethodGet, "/debug/pprof/cmdline/", profCmdline),
		NewRawRouteBuilder(MethodGet, "/debug/pprof/profile/", profProfile),
		NewRawRouteBuilder(MethodGet, "/debug/pprof/symbol/", profSymbol),
		NewRawRouteBuilder(MethodGet, "/debug/pprof/trace/", profTrace),
		NewRawRouteBuilder(MethodGet, "/debug/pprof/heap/", profHeap),
		NewRawRouteBuilder(MethodGet, "/debug/pprof/goroutine/", profGoroutine),
		NewRawRouteBuilder(MethodGet, "/debug/pprof/block/", profBlock),
		NewRawRouteBuilder(MethodGet, "/debug/pprof/threadcreate/", profThreadcreate),
		NewRawRouteBuilder(MethodGet, "/debug/pprof/mutex/", profMutex),
	}
}

func profIndex(w http.ResponseWriter, r *http.Request) {
	pprof.Index(w, r)
}

func pprofAllocsIndex(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("allocs").ServeHTTP(w, r)
}

func profCmdline(w http.ResponseWriter, r *http.Request) {
	pprof.Cmdline(w, r)
}

func profProfile(w http.ResponseWriter, r *http.Request) {
	pprof.Profile(w, r)
}

func profSymbol(w http.ResponseWriter, r *http.Request) {
	pprof.Symbol(w, r)
}

func profTrace(w http.ResponseWriter, r *http.Request) {
	pprof.Trace(w, r)
}

func profHeap(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("heap").ServeHTTP(w, r)
}

func profGoroutine(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("goroutine").ServeHTTP(w, r)
}

func profBlock(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("block").ServeHTTP(w, r)
}

func profThreadcreate(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("threadcreate").ServeHTTP(w, r)
}

func profMutex(w http.ResponseWriter, r *http.Request) {
	pprof.Handler("mutex").ServeHTTP(w, r)
}
