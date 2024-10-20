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

func Setup(cfg *Config) {
	ho := &slog.HandlerOptions{
		AddSource: true,
		Level:     level(cfg.Level),
	}

	var hnd slog.Handler

	if cfg.IsJSON {
		hnd = slog.NewJSONHandler(os.Stderr, ho)
	} else {
		hnd = slog.NewTextHandler(os.Stderr, ho)
	}

	slog.New(hnd.WithAttrs(cfg.Attributes))
}

func level(lvl string) slog.Level {
	lv := slog.LevelVar{}
	if err := lv.UnmarshalText([]byte(lvl)); err != nil {
		return slog.LevelInfo
	}

	return lv.Level()
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

func ErrorAttr(err error) slog.Attr {
	return slog.Any("error", err)
}
