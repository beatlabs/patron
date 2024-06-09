//go:build integration

package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestSetup(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	ctx := context.Background()

	got, err := Setup(ctx, resource.Default())
	assert.NoError(t, err)

	assert.NotNil(t, otel.GetMeterProvider())

	assert.NoError(t, got.Shutdown(ctx))
}
