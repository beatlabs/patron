// Package retry provides a retry pattern implementation.
package retry

import (
	"errors"
	"time"
)

// Action function to execute in retry.
type Action func() (any, error)

// Retry implementation with configurable attempts and optional delay.
type Retry struct {
	attempts int
	delay    time.Duration
}

// New returns a retry instance.
func New(attempts int, delay time.Duration) (*Retry, error) {
	if attempts <= 1 {
		return nil, errors.New("attempts should be greater than 1")
	}

	return &Retry{attempts: attempts, delay: delay}, nil
}

// Execute a specific action with retries.
func (r Retry) Execute(act Action) (any, error) {
	var err error
	var res any

	for range r.attempts {
		res, err = act()
		if err == nil {
			return res, nil
		}

		if r.delay > 0 {
			time.Sleep(r.delay)
		}
	}
	return nil, err
}
