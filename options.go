package patron

import (
	"errors"

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

			svc.logConfig.attrs = append(svc.logConfig.attrs, attr)
		}

		return nil
	}
}

// WithJSONLogger to use Go's slog package.
func WithJSONLogger() OptionFunc {
	return func(svc *Service) error {
		svc.logConfig.json = true
		return nil
	}
}
