package http

import (
	"errors"
	"time"
)

// OptionFunc definition for configuring the component in a functional way.
type OptionFunc func(*Component) error

// WithTLS functional option.
func WithTLS(cert, key string) OptionFunc {
	return func(cmp *Component) error {
		if cert == "" || key == "" {
			return errors.New("cert file or key file was empty")
		}

		cmp.certFile = cert
		cmp.keyFile = key
		return nil
	}
}

// WithReadTimeout functional option.
func WithReadTimeout(rt time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if rt <= 0*time.Second {
			return errors.New("negative or zero read timeout provided")
		}
		cmp.readTimeout = rt
		return nil
	}
}

// WithWriteTimeout functional option.
func WithWriteTimeout(wt time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if wt <= 0*time.Second {
			return errors.New("negative or zero write timeout provided")
		}
		cmp.writeTimeout = wt
		return nil
	}
}

// WithHandlerTimeout functional option.
func WithHandlerTimeout(wt time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if wt <= 0*time.Second {
			return errors.New("negative or zero handler timeout provided")
		}
		cmp.handlerTimeout = wt
		return nil
	}
}

// WithShutdownGracePeriod functional option.
func WithShutdownGracePeriod(gp time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if gp <= 0*time.Second {
			return errors.New("negative or zero shutdown grace period timeout provided")
		}
		cmp.shutdownGracePeriod = gp
		return nil
	}
}

// WithPort functional option.
func WithPort(port int) OptionFunc {
	return func(cmp *Component) error {
		if port <= 0 || port > 65535 {
			return errors.New("invalid HTTP Port provided")
		}
		cmp.port = port
		return nil
	}
}
