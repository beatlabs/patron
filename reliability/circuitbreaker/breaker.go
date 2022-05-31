// Package circuitbreaker provides a circuit breaker pattern implementation.
package circuitbreaker

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// OpenError defines a open error.
type OpenError struct{}

func (oe OpenError) Error() string {
	return "circuit is open"
}

type status int

const (
	close status = iota // nolint:predeclared
	open
)

var (
	tsFuture       = int64(math.MaxInt64)
	errOpen        = new(OpenError)
	breakerCounter *prometheus.CounterVec
	statusMap      = map[status]string{close: "close", open: "open"}
)

func init() {
	breakerCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "reliability",
			Subsystem: "circuit_breaker",
			Name:      "errors",
			Help:      "Circuit breaker status, classified by name and status",
		},
		[]string{"name", "status"},
	)

	prometheus.MustRegister(breakerCounter)
}

func breakerCounterInc(name string, st status) {
	breakerCounter.WithLabelValues(name, statusMap[st]).Inc()
}

// Setting definition.
type Setting struct {
	// The threshold for the circuit to open.
	FailureThreshold uint
	// The timeout after which we set the state to half-open and allow a retry.
	RetryTimeout time.Duration
	// The threshold of the retry successes which returns the state to open.
	RetrySuccessThreshold uint
	// The threshold of how many retry executions are allowed when the status is half-open.
	MaxRetryExecutionThreshold uint
}

// Action function to execute in circuit breaker.
type Action func() (interface{}, error)

// CircuitBreaker implementation.
type CircuitBreaker struct {
	name string
	set  Setting
	sync.RWMutex
	status     status
	executions uint
	failures   uint
	retries    uint
	nextRetry  int64
}

// New constructor.
func New(name string, s Setting) (*CircuitBreaker, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}

	if s.MaxRetryExecutionThreshold < s.RetrySuccessThreshold {
		return nil, errors.New("max retry has to be greater than the retry threshold")
	}

	return &CircuitBreaker{
		name:       name,
		set:        s,
		status:     close,
		executions: 0,
		failures:   0,
		retries:    0,
		nextRetry:  tsFuture,
	}, nil
}

func (cb *CircuitBreaker) isHalfOpen() bool {
	cb.RLock()
	defer cb.RUnlock()
	if cb.status == open && cb.nextRetry <= time.Now().UnixNano() {
		return true
	}
	return false
}

func (cb *CircuitBreaker) isOpen() bool {
	cb.RLock()
	defer cb.RUnlock()
	if cb.status == open && cb.nextRetry > time.Now().UnixNano() {
		return true
	}
	return false
}

func (cb *CircuitBreaker) isClose() bool {
	cb.RLock()
	defer cb.RUnlock()
	return cb.status == close
}

// Execute the function enclosed.
func (cb *CircuitBreaker) Execute(act Action) (interface{}, error) {
	if cb.isOpen() {
		return nil, errOpen
	}

	resp, err := act()
	if err != nil {
		cb.incFailure()
		return nil, err
	}
	cb.incSuccess()

	return resp, err
}

func (cb *CircuitBreaker) incFailure() {
	// allow closed and half open to transition to open
	if cb.isOpen() {
		return
	}
	cb.Lock()
	defer cb.Unlock()

	cb.failures++

	if cb.status == close && cb.failures >= cb.set.FailureThreshold {
		cb.transitionToOpen()
		return
	}

	cb.executions++

	if cb.executions < cb.set.MaxRetryExecutionThreshold {
		return
	}

	cb.transitionToOpen()
}

func (cb *CircuitBreaker) incSuccess() {
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
	cb.transitionToClose()
}

func (cb *CircuitBreaker) transitionToOpen() {
	cb.status = open
	cb.failures = 0
	cb.executions = 0
	cb.retries = 0
	cb.nextRetry = time.Now().Add(cb.set.RetryTimeout).UnixNano()
	breakerCounterInc(cb.name, cb.status)
}

func (cb *CircuitBreaker) transitionToClose() {
	cb.status = close
	cb.failures = 0
	cb.executions = 0
	cb.retries = 0
	cb.nextRetry = tsFuture
	breakerCounterInc(cb.name, cb.status)
}
