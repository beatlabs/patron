package http

import (
	"net/http"
)

// ReadyStatus type.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
type ReadyStatus int

const (
	// Ready represents a state defining a Ready state.
	Ready ReadyStatus = 1
	// NotReady represents a state defining a NotReady state.
	NotReady ReadyStatus = 2

	// ReadyPath of the service.
	ReadyPath = "/ready"
)

// ReadyCheckFunc defines a function type for implementing a readiness check.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
type ReadyCheckFunc func() ReadyStatus

func readyCheckRoute(rcf ReadyCheckFunc) *RouteBuilder {
	f := func(w http.ResponseWriter, r *http.Request) {
		switch rcf() {
		case Ready:
			w.WriteHeader(http.StatusOK)
		case NotReady:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}
	return NewRawRouteBuilder(ReadyPath, f).MethodGet()
}
