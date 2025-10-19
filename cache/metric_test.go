package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestUseCaseAttribute(t *testing.T) {
	assert.Equal(t, attribute.String("cache.use_case", "test"), UseCaseAttribute("test"))
}

func TestSetupAndUseMetrics(t *testing.T) {
	require.NoError(t, SetupMetricsOnce())

	read := metricsdk.NewManualReader()
	provider := metricsdk.NewMeterProvider(metricsdk.WithReader(read))
	defer func() {
		err := provider.Shutdown(context.Background())
		if err != nil {
			t.Fatal(err)
		}
	}()

	otel.SetMeterProvider(provider)

	assert.NotNil(t, cacheCounter)

	ObserveHit(context.Background(), attribute.String("test", "test"))

	collectedMetrics := &metricdata.ResourceMetrics{}
	require.NoError(t, read.Collect(context.Background(), collectedMetrics))

	assert.Len(t, collectedMetrics.ScopeMetrics, 1)

	ObserveMiss(context.Background(), attribute.String("test", "test"))

	collectedMetrics = &metricdata.ResourceMetrics{}
	require.NoError(t, read.Collect(context.Background(), collectedMetrics))

	assert.Len(t, collectedMetrics.ScopeMetrics, 1)
}
