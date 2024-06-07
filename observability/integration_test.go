//go:build integration

package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	ctx := context.Background()

	got, err := Setup(ctx, "test", "1.2.3")
	assert.NoError(t, err)

	assert.NoError(t, got.Shutdown(ctx))
}
