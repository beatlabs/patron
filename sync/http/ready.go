package http

import (
	"net/http"
)

// ReadyStatus type.
type ReadyStatus int

const (
	// Ready represents a state defining a Ready state.
	Ready ReadyStatus = 1
	// NotReady represents a state defining a NotReady state.
	NotReady ReadyStatus = 2
)

// ReadyCheckFunc defines a function type for implementing a readiness check.
type ReadyCheckFunc func() ReadyStatus

func readyCheckRoute(rcf ReadyCheckFunc) Route {

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
	return NewRouteRaw("/ready", http.MethodGet, f, false)
}
