package http

import (
	"net/http"
	"net/http/pprof"
)

func profilingRoutes() []*RouteBuilder {
	return []*RouteBuilder{
		NewRawRouteBuilder("/debug/pprof/", profIndex).MethodGet(),
		NewRawRouteBuilder("/debug/pprof/allocs/", pprofAllocsIndex).MethodGet(),
		NewRawRouteBuilder("/debug/pprof/cmdline/", profCmdline).MethodGet(),
		NewRawRouteBuilder("/debug/pprof/profile/", profProfile).MethodGet(),
		NewRawRouteBuilder("/debug/pprof/symbol/", profSymbol).MethodGet(),
		NewRawRouteBuilder("/debug/pprof/trace/", profTrace).MethodGet(),
		NewRawRouteBuilder("/debug/pprof/heap/", profHeap).MethodGet(),
		NewRawRouteBuilder("/debug/pprof/goroutine/", profGoroutine).MethodGet(),
		NewRawRouteBuilder("/debug/pprof/block/", profBlock).MethodGet(),
		NewRawRouteBuilder("/debug/pprof/threadcreate/", profThreadcreate).MethodGet(),
		NewRawRouteBuilder("/debug/pprof/mutex/", profMutex).MethodGet(),
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
