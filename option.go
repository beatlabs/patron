package patron

import (
	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/sync/http"
)

// OptionFunc definition for configuring the service in a functional way.
type OptionFunc func(*Service) error

// Routes option for adding routes to the default HTTP component.
func Routes(rr []http.Route) OptionFunc {
	return func(s *Service) error {
		if len(rr) == 0 {
			return errors.New("routes are required")
		}
		s.routes = rr
		log.Info("routes options are set")
		return nil
	}
}

// Middlewares option for adding generic middlewares to the default HTTP component.
func Middlewares(mm ...http.MiddlewareFunc) OptionFunc {
	return func(s *Service) error {
		if len(mm) == 0 {
			return errors.New("middlewares are required")
		}
		s.middlewares = mm
		log.Info("middleware options are set")
		return nil
	}
}

// HealthCheck option for overriding the default health check of the default HTTP component.
func HealthCheck(hcf http.HealthCheckFunc) OptionFunc {
	return func(s *Service) error {
		if hcf == nil {
			return errors.New("health check func is required")
		}
		s.hcf = hcf
		log.Info("health check func is set")
		return nil
	}
}

// Components option for adding additional components to the service.
func Components(cc ...Component) OptionFunc {
	return func(s *Service) error {
		if len(cc) == 0 || cc[0] == nil {
			return errors.New("components are required")
		}
		s.cps = append(s.cps, cc...)
		log.Info("component options are set")
		return nil
	}
}

// SIGHUP option for adding a handler when the service receives a SIGHUP.
func SIGHUP(handler func()) OptionFunc {
	return func(s *Service) error {
		if handler == nil {
			return errors.New("handler is nil")
		}
		s.sighupHandler = handler
		log.Info("SIGHUP handler set")
		return nil
	}
}
