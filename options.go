package patron

import (
	"errors"
	"net/http"
	"os"

	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/log/std"
)

type OptionFunc func(svc *Service) error

// WithRouter replaces the default v1 HTTP component with a new component v2 based on http.Handler.
func WithRouter(handler http.Handler) OptionFunc {
	return func(svc *Service) error {
		if handler == nil {
			return errors.New("provided router is nil")
		}

		log.Debug("router will be used with the v2 HTTP component")
		svc.httpRouter = handler

		return nil
	}
}

// WithComponents adds custom components to the Patron Service.
func WithComponents(cc ...Component) OptionFunc {
	return func(svc *Service) error {
		if len(cc) == 0 {
			return errors.New("provided components slice was empty")
		}

		log.Debug("setting components")
		svc.cps = append(svc.cps, cc...)

		return nil
	}
}

// WithSIGHUP adds a custom handler for handling WithSIGHUP.
func WithSIGHUP(handler func()) OptionFunc {
	return func(svc *Service) error {
		if handler == nil {
			return errors.New("provided WithSIGHUP handler was nil")
		}

		log.Debug("setting WithSIGHUP handler func")
		svc.sighupHandler = handler

		return nil
	}
}

// WithLogFields options to pass in additional log fields.
func WithLogFields(fields map[string]interface{}) OptionFunc {
	return func(svc *Service) error {
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

// WithLogger to pass in custom logger.
func WithLogger(logger log.Logger) OptionFunc {
	return func(svc *Service) error {
		svc.config.logger = logger

		return nil
	}
}

// WithTextLogger to use Go's standard logger.
func WithTextLogger() OptionFunc {
	return func(svc *Service) error {
		svc.config.logger = std.New(os.Stderr, getLogLevel(), svc.config.fields)

		return nil
	}
}
