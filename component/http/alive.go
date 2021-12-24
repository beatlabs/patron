package http

import (
	"net/http"
)

// AliveStatus type representing the liveness of the service via HTTP component.
type AliveStatus int

const (
	// Alive represents a state defining a Alive state.
	Alive AliveStatus = 1
	// Unresponsive represents a state defining a Unresponsive state.
	Unresponsive AliveStatus = 2
)

// AliveCheckFunc defines a function type for implementing a liveness check.
type AliveCheckFunc func() AliveStatus

func aliveCheckRoute(acf AliveCheckFunc) *RouteBuilder {
	f := func(w http.ResponseWriter, r *http.Request) {
		switch acf() {
		case Alive:
			w.WriteHeader(http.StatusOK)
		case Unresponsive:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}
	return NewRawRouteBuilder("/alive", f).MethodGet()
}
