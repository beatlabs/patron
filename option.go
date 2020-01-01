package patron

import (
	"errors"

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

// AliveCheck option for overriding the default liveness check of the default HTTP component.
func AliveCheck(acf http.AliveCheckFunc) OptionFunc {
	return func(s *Service) error {
		if acf == nil {
			return errors.New("alive check func is required")
		}
		s.acf = acf
		log.Info("alive check func is set")
		return nil
	}
}

// ReadyCheck option for overriding the default readiness check of the default HTTP component.
func ReadyCheck(rcf http.ReadyCheckFunc) OptionFunc {
	return func(s *Service) error {
		if rcf == nil {
			return errors.New("ready check func is required")
		}
		s.rcf = rcf
		log.Info("ready check func is set")
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
