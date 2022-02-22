package v2

import (
	"errors"
	"time"
)

// OptionFunc definition for configuring the component in a functional way.
type OptionFunc func(*Component) error

// TLS functional option.
func TLS(cert, key string) OptionFunc {
	return func(cmp *Component) error {
		if cert == "" || key == "" {
			return errors.New("cert file or key file was empty")
		}

		cmp.certFile = cert
		cmp.keyFile = key
		return nil
	}
}

// ReadTimeout functional option.
func ReadTimeout(rt time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if rt <= 0*time.Second {
			return errors.New("negative or zero read timeout provided")
		}
		cmp.readTimeout = rt
		return nil
	}
}

// WriteTimeout functional option.
func WriteTimeout(wt time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if wt <= 0*time.Second {
			return errors.New("negative or zero write timeout provided")
		}
		cmp.writeTimeout = wt
		return nil
	}
}

// HandlerTimeout functional option.
func HandlerTimeout(wt time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if wt <= 0*time.Second {
			return errors.New("negative or zero handler timeout provided")
		}
		cmp.handlerTimeout = wt
		return nil
	}
}

// ShutdownGracePeriod functional option.
func ShutdownGracePeriod(gp time.Duration) OptionFunc {
	return func(cmp *Component) error {
		if gp <= 0*time.Second {
			return errors.New("negative or zero shutdown grace period timeout provided")
		}
		cmp.shutdownGracePeriod = gp
		return nil
	}
}

// Port functional option.
func Port(port int) OptionFunc {
	return func(cmp *Component) error {
		if port <= 0 || port > 65535 {
			return errors.New("invalid HTTP Port provided")
		}
		cmp.port = port
		return nil
	}
}
