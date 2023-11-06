package http

import (
	"log/slog"
	"net/http"

	"github.com/beatlabs/patron/log"
)

// AliveStatus type representing the liveness of the service via HTTP component.
type AliveStatus int

// ReadyStatus type.
type ReadyStatus int

const (
	// Alive represents a state defining an Alive state.
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
func LivenessCheckRoute(acf LivenessCheckFunc) (*Route, error) {
	f := func(w http.ResponseWriter, r *http.Request) {
		val := acf()
		switch val {
		case Alive:
			w.WriteHeader(http.StatusOK)
		case Unhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
			log.FromContext(r.Context()).Error("wrong live check status returned", slog.Int("status", int(val)))
		}
	}

	return NewRoute(http.MethodGet, AlivePath, f)
}

// ReadyCheckRoute returns a route for ready checks.
func ReadyCheckRoute(rcf ReadyCheckFunc) (*Route, error) {
	f := func(w http.ResponseWriter, r *http.Request) {
		val := rcf()
		switch val {
		case Ready:
			w.WriteHeader(http.StatusOK)
		case NotReady:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
			log.FromContext(r.Context()).Error("wrong ready check status returned", slog.Int("status", int(val)))
		}
	}

	return NewRoute(http.MethodGet, ReadyPath, f)
}
