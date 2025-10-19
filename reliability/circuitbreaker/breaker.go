// Package circuitbreaker provides a circuit breaker pattern implementation.
package circuitbreaker

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"

	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// OpenError definition for the open state.
type OpenError struct{}

func (oe OpenError) Error() string {
	return "circuit is open"
}

type status int

const (
	packageName     = "circuit-breaker"
	statusAttribute = "status"

	closed status = iota
	opened
)

var (
	tsFuture   = int64(math.MaxInt64)
	errOpen    = new(OpenError)
	openedAttr = attribute.Int64(statusAttribute, int64(opened))
	closedAttr = attribute.Int64(statusAttribute, int64(closed))
)

func (cb *CircuitBreaker) breakerCounterInc(ctx context.Context, st status) {
	stateAttr := closedAttr
	switch st {
	case opened:
		stateAttr = openedAttr
	case closed:
		stateAttr = closedAttr
	}
	cb.statusCounter.Add(ctx, 1, metric.WithAttributes(stateAttr, attribute.String("name", cb.name)))
}

// Setting definition.
type Setting struct {
	// FailureThreshold for the circuit to open.
	FailureThreshold uint
	// RetryTimeout after which we set the state to half-open and allow a retry.
	RetryTimeout time.Duration
	// RetrySuccessThreshold which returns the state to open.
	RetrySuccessThreshold uint
	// MaxRetryExecutionThreshold which defines how many tries are allowed when the status is half-open.
	MaxRetryExecutionThreshold uint
}

// Action function to execute in circuit breaker.
type Action func() (any, error)

// CircuitBreaker implementation.
type CircuitBreaker struct {
	name          string
	set           Setting
	statusCounter metric.Int64Counter
	sync.RWMutex
	status     status
	executions uint
	failures   uint
	retries    uint
	nextRetry  int64
}

// New returns a new instance.
func New(name string, s Setting) (*CircuitBreaker, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}

	if s.MaxRetryExecutionThreshold < s.RetrySuccessThreshold {
		return nil, errors.New("max retry has to be greater than the retry threshold")
	}

	counter, err := patronmetric.Int64Counter(packageName, "circuit-breaker.status", "Circuit breaker status counter.", "1")
	if err != nil {
		return nil, err
	}

	return &CircuitBreaker{
		name:          name,
		set:           s,
		statusCounter: counter,
		status:        closed,
		executions:    0,
		failures:      0,
		retries:       0,
		nextRetry:     tsFuture,
	}, nil
}

func (cb *CircuitBreaker) isHalfOpen() bool {
	cb.RLock()
	defer cb.RUnlock()
	if cb.status == opened && cb.nextRetry <= time.Now().UnixNano() {
		return true
	}
	return false
}

func (cb *CircuitBreaker) isOpen() bool {
	cb.RLock()
	defer cb.RUnlock()
	if cb.status == opened && cb.nextRetry > time.Now().UnixNano() {
		return true
	}
	return false
}

func (cb *CircuitBreaker) isClose() bool {
	cb.RLock()
	defer cb.RUnlock()
	return cb.status == closed
}

// Execute the provided action.
func (cb *CircuitBreaker) Execute(ctx context.Context, act Action) (any, error) {
	if cb.isOpen() {
		return nil, errOpen
	}

	resp, err := act()
	if err != nil {
		cb.incFailure(ctx)
		return nil, err
	}
	cb.incSuccess(ctx)

	return resp, err
}

func (cb *CircuitBreaker) incFailure(ctx context.Context) {
	// allow closed and half open to transition to open
	if cb.isOpen() {
		return
	}
	cb.Lock()
	defer cb.Unlock()

	cb.failures++

	if cb.status == closed && cb.failures >= cb.set.FailureThreshold {
		cb.transitionToOpen(ctx)
		return
	}

	cb.executions++

	if cb.executions < cb.set.MaxRetryExecutionThreshold {
		return
	}

	cb.transitionToOpen(ctx)
}

func (cb *CircuitBreaker) incSuccess(ctx context.Context) {
	// allow only half open in order to transition to closed
	if !cb.isHalfOpen() {
		return
	}
	cb.Lock()
	defer cb.Unlock()

	cb.retries++
	cb.executions++

	if cb.retries < cb.set.RetrySuccessThreshold {
		return
	}
	cb.transitionToClose(ctx)
}

func (cb *CircuitBreaker) transitionToOpen(ctx context.Context) {
	cb.status = opened
	cb.failures = 0
	cb.executions = 0
	cb.retries = 0
	cb.nextRetry = time.Now().Add(cb.set.RetryTimeout).UnixNano()
	cb.breakerCounterInc(ctx, cb.status)
}

func (cb *CircuitBreaker) transitionToClose(ctx context.Context) {
	cb.status = closed
	cb.failures = 0
	cb.executions = 0
	cb.retries = 0
	cb.nextRetry = tsFuture
	cb.breakerCounterInc(ctx, cb.status)
}
