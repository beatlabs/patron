package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type (
	ShutdownFunc       func()
	CollectMetricsFunc func(expected int) *metricdata.ResourceMetrics
)

// SetupMetrics sets up the metrics provider and reader for testing.
func SetupMetrics(ctx context.Context, t *testing.T) (ShutdownFunc, CollectMetricsFunc) {
	read := metricsdk.NewManualReader()

	provider := metricsdk.NewMeterProvider(metricsdk.WithReader(read))
	otel.SetMeterProvider(provider)

	shutdownFunc := func() {
		require.NoError(t, provider.Shutdown(ctx))
	}
	collectMetrics := func(expected int) *metricdata.ResourceMetrics {
		cm := &metricdata.ResourceMetrics{}
		require.NoError(t, read.Collect(ctx, cm))
		require.GreaterOrEqual(t, len(cm.ScopeMetrics), 1)
		require.GreaterOrEqual(t, len(AllMetrics(cm)), expected)
		return cm
	}
	return shutdownFunc, collectMetrics
}

// AllMetrics returns a flattened slice of all metrics across all scopes
// in the given ResourceMetrics.
func AllMetrics(rm *metricdata.ResourceMetrics) []metricdata.Metrics {
	var all []metricdata.Metrics
	for _, sm := range rm.ScopeMetrics {
		all = append(all, sm.Metrics...)
	}
	return all
}
