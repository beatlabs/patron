package http

import (
	"net/http"
)

// AliveStatus type representing the liveness of the service via HTTP component.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
type AliveStatus int

const (
	// Alive represents a state defining an Alive state.
	Alive AliveStatus = 1
	// Unresponsive represents a state defining a Unresponsive state.
	Unresponsive AliveStatus = 2

	// AlivePath of the service.
	AlivePath = "/alive"
)

// AliveCheckFunc defines a function type for implementing a liveness check.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
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
	return NewRawRouteBuilder(AlivePath, f).MethodGet()
}
