// Package correlation provides support for correlation id's and propagation.
package correlation

import (
	"context"

	"github.com/google/uuid"
)

const (
	// HeaderID constant.
	HeaderID string = "X-Correlation-Id"
	// ID constant.
	ID string = "correlationID"
)

type idContextKey struct{}

var idKey = idContextKey{}

// IDFromContext returns the correlation ID from the context.
// If no ID is set a new one is generated.
func IDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(idKey).(string); ok {
		return id
	}
	return uuid.New().String()
}

// ContextWithID sets a correlation ID to a context.
func ContextWithID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, idKey, correlationID)
}
