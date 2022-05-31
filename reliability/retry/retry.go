// Package retry provides a retry pattern implementation.
package retry

import (
	"errors"
	"time"
)

// Action function to execute in retry.
type Action func() (interface{}, error)

// Retry pattern with attempts and optional delay.
type Retry struct {
	attempts int
	delay    time.Duration
}

// New constructor.
func New(attempts int, delay time.Duration) (*Retry, error) {
	if attempts <= 1 {
		return nil, errors.New("attempts should be greater than 1")
	}
	return &Retry{attempts: attempts, delay: delay}, nil
}

// Execute a specific action.
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
