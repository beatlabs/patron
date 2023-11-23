// Package log provides logging abstractions.
package log

import (
	"context"
	"log/slog"
)

type ctxKey struct{}

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
