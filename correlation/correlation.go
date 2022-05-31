// Package correlation provides support for correlation id's and propagation.
package correlation

import (
	"context"
	"net/http"

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

func GetOrSetHeaderID(h http.Header) string {
	cor, ok := h[HeaderID]
	if !ok {
		corID := uuid.New().String()
		h.Set(HeaderID, corID)
		return corID
	}
	if len(cor) == 0 {
		corID := uuid.New().String()
		h.Set(HeaderID, corID)
		return corID
	}
	if cor[0] == "" {
		corID := uuid.New().String()
		h.Set(HeaderID, corID)
		return corID
	}
	return cor[0]
}
