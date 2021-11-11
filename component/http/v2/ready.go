package v2

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

var defaultReadyCheck = func() ReadyStatus { return Ready }
