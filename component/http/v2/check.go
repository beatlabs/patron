package v2

import (
	"net/http"
)

// AliveStatus type representing the liveness of the service via HTTP component.
type AliveStatus int

// ReadyStatus type.
type ReadyStatus int

const (
	// Alive represents a state defining a Alive state.
	Alive AliveStatus = 1
	// Unhealthy represents an unhealthy alive state.
	Unhealthy AliveStatus = 2

	// Ready represents a state defining a Ready state.
	Ready ReadyStatus = 1
	// NotReady represents a state defining a NotReady state.
	NotReady ReadyStatus = 2

	// AlivePath of the component.
	AlivePath = "/alive"
	// ReadyPath of the component.
	ReadyPath = "/ready"
)

// ReadyCheckFunc defines a function type for implementing a readiness check.
type ReadyCheckFunc func() ReadyStatus

// LivenessCheckFunc defines a function type for implementing a liveness check.
type LivenessCheckFunc func() AliveStatus

// LivenessCheckRoute returns a route for liveness checks.
func LivenessCheckRoute(acf LivenessCheckFunc) *Route {
	f := func(w http.ResponseWriter, r *http.Request) {
		switch acf() {
		case Alive:
			w.WriteHeader(http.StatusOK)
		case Unhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}
	route, _ := NewRoute(http.MethodGet, AlivePath, f)
	return route
}

// ReadyCheckRoute returns a route for ready checks.
func ReadyCheckRoute(rcf ReadyCheckFunc) *Route {
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
	route, _ := NewRoute(http.MethodGet, ReadyPath, f)
	return route
}
