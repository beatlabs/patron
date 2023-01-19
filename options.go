package patron

import (
	"errors"
	"os"

	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/log/std"
)

type OptionFunc func(svc *Service) error

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
