//go:build integration

package metric

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestSetup(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	ctx := context.Background()

	got, err := Setup(ctx, resource.Default())
	require.NoError(t, err)

	assert.NotNil(t, otel.GetMeterProvider())

	require.NoError(t, got.ForceFlush(ctx))

	require.NoError(t, got.Shutdown(ctx))
}
