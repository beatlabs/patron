package http

import (
	"errors"
	"time"
)

// OptionFunc configures the HTTP Component.
type OptionFunc func(*Component) error

// WithTLS enables HTTPS using the provided certificate and key file paths.
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

// WithReadTimeout sets the server read timeout.
func WithReadTimeout(rt time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if rt <= 0*time.Second {
			return errors.New("negative or zero read timeout provided")
		}
		cmp.readTimeout = rt
		return nil
	}
}

// WithWriteTimeout sets the server write timeout.
func WithWriteTimeout(wt time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if wt <= 0*time.Second {
			return errors.New("negative or zero write timeout provided")
		}
		cmp.writeTimeout = wt
		return nil
	}
}

// WithHandlerTimeout sets the per-request handler timeout.
func WithHandlerTimeout(wt time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if wt <= 0*time.Second {
			return errors.New("negative or zero handler timeout provided")
		}
		cmp.handlerTimeout = wt
		return nil
	}
}

// WithShutdownGracePeriod sets the graceful shutdown timeout.
func WithShutdownGracePeriod(gp time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if gp <= 0*time.Second {
			return errors.New("negative or zero shutdown grace period timeout provided")
		}
		cmp.shutdownGracePeriod = gp
		return nil
	}
}

// WithPort overrides the listening port.
func WithPort(port int) OptionFunc {
	return func(cmp *Component) error {
		if port <= 0 || port > 65535 {
			return errors.New("invalid HTTP Port provided")
		}
		cmp.port = port
		return nil
	}
}
