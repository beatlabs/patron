// Package retry provides a retry pattern implementation.
package retry

import (
	"errors"
	"time"
)

// Action function to execute in retry.
type Action func() (interface{}, error)

// Retry pattern with retries and optional delay.
type Retry struct {
	retries int
	delay   time.Duration
}

// New constructor.
func New(retries int, delay time.Duration) (*Retry, error) {
	if retries < 0 {
		return nil, errors.New("retries should be zero or positive")
	}
	return &Retry{retries: retries, delay: delay}, nil
}

// Execute a specific action.
func (r Retry) Execute(act Action) (interface{}, error) {
	current := r.retries
	var err error
	var res interface{}

	for {
		res, err = act()
		if err == nil {
			return res, nil
		}
		current--
		if current == 0 {
			break
		}

		if r.delay > 0 {
			time.Sleep(r.delay)
		}
	}
	return nil, err
}
