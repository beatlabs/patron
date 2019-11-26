package http

import (
	"time"

	"github.com/beatlabs/patron/errors"
)

// OptionFunc defines a option func for the HTTP component.
type OptionFunc func(*Component) error

// Port option for setting the ports of the HTTP component.
func Port(port int) OptionFunc {
	return func(s *Component) error {
		if port <= 0 || port > 65535 {
			return errors.New("invalid port")
		}
		s.httpPort = port
		return nil
	}
}

// Timeouts option for setting the timeouts of the HTTP component.
func Timeouts(read, write time.Duration) OptionFunc {
	return func(s *Component) error {
		s.httpReadTimeout = read
		s.httpWriteTimeout = write
		return nil
	}
}

// Routes option for setting the routes of the HTTP component.
func Routes(rr []Route) OptionFunc {
	return func(s *Component) error {
		if len(rr) == 0 {
			return errors.New("routes are empty")
		}
		s.routes = append(s.routes, rr...)
		return nil
	}
}

// Middlewares option for setting the routes middlewares of the HTTP component.
func Middlewares(mm ...MiddlewareFunc) OptionFunc {
	return func(s *Component) error {
		if len(mm) == 0 {
			return errors.New("middlewares are empty")
		}
		s.middlewares = append(s.middlewares, mm...)
		return nil
	}
}

// AliveCheck option for setting the liveness check function of the HTTP component.
func AliveCheck(acf AliveCheckFunc) OptionFunc {
	return func(s *Component) error {
		if acf == nil {
			return errors.New("alive check function is not defined")
		}
		s.ac = acf
		return nil
	}
}

// ReadyCheck option for setting the readiness check function of the HTTP component.
func ReadyCheck(rcf ReadyCheckFunc) OptionFunc {
	return func(s *Component) error {
		if rcf == nil {
			return errors.New("alive check function is not defined")
		}
		s.rc = rcf
		return nil
	}
}

// Secure option for securing the default HTTP component.
func Secure(certFile, keyFile string) OptionFunc {
	return func(s *Component) error {
		if certFile == "" {
			return errors.New("cert file is required")
		}
		if keyFile == "" {
			return errors.New("key file is required")
		}
		s.certFile = certFile
		s.keyFile = keyFile
		return nil
	}
}
