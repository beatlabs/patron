package http

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/beatlabs/patron/reliability/circuitbreaker"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
)

// OptionFunc definition for configuring the client in a functional way.
type OptionFunc func(*TracedClient) error

// WithTimeout option for adjusting the timeout of the connection.
func WithTimeout(timeout time.Duration) OptionFunc {
	return func(tc *TracedClient) error {
		if timeout <= 0 {
			return errors.New("timeout must be positive")
		}
		tc.cl.Timeout = timeout
		return nil
	}
}

// WithCircuitBreaker option for setting up a circuit breaker.
func WithCircuitBreaker(name string, set circuitbreaker.Setting) OptionFunc {
	return func(tc *TracedClient) error {
		cb, err := circuitbreaker.New(name, set)
		if err != nil {
			return fmt.Errorf("failed to set circuit breaker: %w", err)
		}
		tc.cb = cb
		return nil
	}
}

// WithTransport option for setting the WithTransport for the client.
func WithTransport(rt http.RoundTripper) OptionFunc {
	return func(tc *TracedClient) error {
		if rt == nil {
			return errors.New("transport must be supplied")
		}
		tc.cl.Transport = &nethttp.Transport{RoundTripper: rt}
		return nil
	}
}

// WithCheckRedirect option for setting the WithCheckRedirect for the client.
func WithCheckRedirect(cr func(req *http.Request, via []*http.Request) error) OptionFunc {
	return func(tc *TracedClient) error {
		if cr == nil {
			return errors.New("check redirect must be supplied")
		}
		tc.cl.CheckRedirect = cr
		return nil
	}
}
