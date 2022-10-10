package patron

import (
	"errors"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/log/std"
	"net/http"
	"os"
)

type OptionFunc func(svc *service) error

// Router replaces the default v1 HTTP component with a new component v2 based on http.Handler.
func Router(handler http.Handler) OptionFunc {
	return func(svc *service) error {
		if handler == nil {
			svc.errors = append(svc.errors, errors.New("provided router is nil"))
		} else {
			svc.httpRouter = handler
			log.Debug("router will be used with the v2 HTTP component")
		}

		return nil
	}
}

// Components adds custom components to the Patron service.
func Components(cc ...Component) OptionFunc {
	return func(svc *service) error {
		if len(cc) == 0 {
			svc.errors = append(svc.errors, errors.New("provided components slice was empty"))
		} else {
			log.Debug("setting components")
			svc.cps = append(svc.cps, cc...)
		}

		return nil
	}
}

// SIGHUP adds a custom handler for handling SIGHUP.
func SIGHUP(handler func()) OptionFunc {
	return func(svc *service) error {
		if handler == nil {
			svc.errors = append(svc.errors, errors.New("provided SIGHUP handler was nil"))
		} else {
			log.Debug("setting SIGHUP handler func")
			svc.sighupHandler = handler
		}

		return nil
	}
}

// LogFields options to pass in additional log fields.
func LogFields(fields map[string]interface{}) OptionFunc {
	return func(svc *service) error {
		for k, v := range fields {
			if k == srv || k == ver || k == host {
				// don't override
				continue
			}
			svc.config.fields[k] = v
		}

		return nil
	}
}

// Logger to pass in custom logger.
func Logger(logger log.Logger) OptionFunc {
	return func(svc *service) error {
		svc.config.logger = logger

		return nil
	}
}

// TextLogger to use Go's standard logger.
func TextLogger() OptionFunc {
	return func(svc *service) error {
		svc.config.logger = std.New(os.Stderr, getLogLevel(), svc.config.fields)

		return nil
	}
}
