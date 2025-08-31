package patron

import (
	"errors"
	"log/slog"
)

// OptionFunc configures the Service.
type OptionFunc func(svc *Service) error

// WithSIGHUP registers a handler invoked when SIGHUP is received.
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

// WithLogFields adds structured logging attributes to the default logger.
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

// WithJSONLogger enables JSON output for slog.
func WithJSONLogger() OptionFunc {
	return func(svc *Service) error {
		svc.observabilityCfg.LogConfig.IsJSON = true
		return nil
	}
}
