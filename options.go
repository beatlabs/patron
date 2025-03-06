package patron

import (
	"errors"
	"log/slog"
	"time"
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
func WithLogFields(attrs ...slog.Attr) OptionFunc {
	return func(svc *Service) error {
		if len(attrs) == 0 {
			return errors.New("attributes are empty")
		}

		for _, attr := range attrs {
			if attr.Key == srv || attr.Key == ver || attr.Key == host {
				// don't override
				continue
			}

			svc.observabilityCfg.LogConfig.Attributes = append(svc.observabilityCfg.LogConfig.Attributes, attr)
		}

		return nil
	}
}

// WithJSONLogger to use Go's slog package.
func WithJSONLogger() OptionFunc {
	return func(svc *Service) error {
		svc.observabilityCfg.LogConfig.IsJSON = true
		return nil
	}
}

// WithShutdownTimeout sets the shutdown timeout for the service.
func WithShutdownTimeout(timeout time.Duration) OptionFunc {
	return func(svc *Service) error {
		if timeout <= 0 {
			return errors.New("shutdown timeout must be positive")
		}
		svc.shutdownTimeout = timeout
		return nil
	}
}
