package httprouter

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
	// Unresponsive represents a state defining a Unresponsive state.
	Unresponsive AliveStatus = 2

	alivePath = "/alive"

	// Ready represents a state defining a Ready state.
	Ready ReadyStatus = 1
	// NotReady represents a state defining a NotReady state.
	NotReady ReadyStatus = 2

	readyPath = "/ready"
)

// ReadyCheckFunc defines a function type for implementing a readiness check.
type ReadyCheckFunc func() ReadyStatus

// AliveCheckFunc defines a function type for implementing a liveness check.
type AliveCheckFunc func() AliveStatus

func aliveCheckRoute(acf AliveCheckFunc) *Route {
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
	route, _ := NewRoute(http.MethodGet, alivePath, f)
	return route
}

func readyCheckRoute(rcf ReadyCheckFunc) *Route {
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
	route, _ := NewRoute(http.MethodGet, readyPath, f)
	return route
}
