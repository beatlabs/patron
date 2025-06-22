// Package log provides logging abstractions.
package log

import (
	"context"
	"log/slog"
	"os"
)

// Config represents the configuration for setting up the logger.
type Config struct {
	Attributes []slog.Attr
	IsJSON     bool
	Level      string
}

type ctxKey struct{}

var logCfg *Config

// Setup sets up the logger with the given configuration.
func Setup(cfg *Config) error {
	logCfg = cfg
	return setDefaultLogger(cfg)
}

// SetLevel sets the logger level.
func SetLevel(lvl string) error {
	logCfg.Level = lvl
	return setDefaultLogger(logCfg)
}

func setDefaultLogger(cfg *Config) error {
	lvl, err := level(cfg.Level)
	if err != nil {
		return err
	}

	ho := &slog.HandlerOptions{
		AddSource: true,
		Level:     lvl,
	}

	var hnd slog.Handler

	if cfg.IsJSON {
		hnd = slog.NewJSONHandler(os.Stderr, ho)
	} else {
		hnd = slog.NewTextHandler(os.Stderr, ho)
	}

	slog.SetDefault(slog.New(hnd.WithAttrs(cfg.Attributes)))
	return nil
}

func level(lvl string) (slog.Level, error) {
	lv := slog.LevelVar{}
	if err := lv.UnmarshalText([]byte(lvl)); err != nil {
		return slog.LevelInfo, err
	}

	return lv.Level(), nil
}

// FromContext returns the logger, if it exists in the context, or nil.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		if l == nil {
			return slog.Default()
		}
		return l
	}
	return slog.Default()
}

// WithContext associates a logger to a context.
func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// Enabled returns true for the appropriate level otherwise false.
func Enabled(l slog.Level) bool {
	return slog.Default().Handler().Enabled(context.Background(), l)
}

// ErrorAttr returns a slog attribute for an error.
func ErrorAttr(err error) slog.Attr {
	return slog.Any("error", err)
}
