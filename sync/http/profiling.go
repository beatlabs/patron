package http

import (
	"net/http"
	"net/http/pprof"
)

func profilingRoutes() []*RouteBuilder {
	return []*RouteBuilder{
		NewRawRouteBuilder("/debug/pprof/", profIndex).WithMethodGet(),
		NewRawRouteBuilder("/debug/pprof/allocs/", pprofAllocsIndex).WithMethodGet(),
		NewRawRouteBuilder("/debug/pprof/cmdline/", profCmdline).WithMethodGet(),
		NewRawRouteBuilder("/debug/pprof/profile/", profProfile).WithMethodGet(),
		NewRawRouteBuilder("/debug/pprof/symbol/", profSymbol).WithMethodGet(),
		NewRawRouteBuilder("/debug/pprof/trace/", profTrace).WithMethodGet(),
		NewRawRouteBuilder("/debug/pprof/heap/", profHeap).WithMethodGet(),
		NewRawRouteBuilder("/debug/pprof/goroutine/", profGoroutine).WithMethodGet(),
		NewRawRouteBuilder("/debug/pprof/block/", profBlock).WithMethodGet(),
		NewRawRouteBuilder("/debug/pprof/threadcreate/", profThreadcreate).WithMethodGet(),
		NewRawRouteBuilder("/debug/pprof/mutex/", profMutex).WithMethodGet(),
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
