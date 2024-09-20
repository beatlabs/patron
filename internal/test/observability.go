package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
)

// SetupMetrics sets up the metrics provider and reader for testing.
func SetupMetrics(t *testing.T) (func(), *metricsdk.ManualReader) {
	read := metricsdk.NewManualReader()
	provider := metricsdk.NewMeterProvider(metricsdk.WithReader(read))
	otel.SetMeterProvider(provider)

	return func() {
		require.NoError(t, provider.Shutdown(context.Background()))
	}, read
}
