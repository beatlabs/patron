// Package retry provides a retry pattern implementation.
package retry

import (
	"errors"
	"time"
)

// Action function to execute in retry.
type Action func() (interface{}, error)

// Retry implementation with configurable attempts and and optional delay.
type Retry struct {
	attempts int
	delay    time.Duration
}

// New returns a retry instance.
func New(attempts int, delay time.Duration) (*Retry, error) {
	if attempts <= 1 {
		return nil, errors.New("attempts should be greater than 1")
	}
	if delay <= 0 {
		return nil, errors.New("delay should be greater than 0")
	}

	return &Retry{attempts: attempts, delay: delay}, nil
}

// Execute a specific action with retries.
func (r Retry) Execute(act Action) (interface{}, error) {
	var err error
	var res interface{}

	for i := 0; i < r.attempts; i++ {
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
