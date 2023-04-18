package patron

import (
	"errors"
	"os"

	"golang.org/x/exp/slog"
)

type OptionFunc func(svc *Service) error

// WithSIGHUP adds a custom handler for handling WithSIGHUP.
func WithSIGHUP(handler func()) OptionFunc {
	return func(svc *Service) error {
		if handler == nil {
			return errors.New("provided WithSIGHUP handler was nil")
		}

		slog.Debug("setting WithSIGHUP handler func")
		svc.sighupHandler = handler

		return nil
	}
}

// WithLogFields options to pass in additional log fields.
func WithLogFields(fields ...slog.Attr) OptionFunc {
	return func(svc *Service) error {
		if len(fields) == 0 {
			return errors.New("fields are empty")
		}

		for _, field := range fields {
			if field.Key == srv || field.Key == ver || field.Key == host {
				// don't override
				continue
			}

			svc.config.fields = append(svc.config.fields, field)
		}

		return nil
	}
}

// WithLogger to pass in custom logger.
func WithLogger(logger *slog.Logger) OptionFunc {
	return func(svc *Service) error {
		svc.config.logger = logger
		return nil
	}
}

// WithJSONLogger to use Go's slog package.
func WithJSONLogger() OptionFunc {
	return func(svc *Service) error {
		svc.config.logger = slog.New(slog.NewJSONHandler(os.Stderr))
		return nil
	}
}
