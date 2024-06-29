package cache

import (
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestUseCaseAttribute(t *testing.T) {
	assert.Equal(t, attribute.String("cache.use_case", "test"), UseCaseAttribute("test"))
}

func TestSetupAndUseMetrics(t *testing.T) {
	SetupMetricsOnce()

	read := metricsdk.NewManualReader()
	provider := metricsdk.NewMeterProvider(metricsdk.WithReader(read))
	defer func() {
		err := provider.Shutdown(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	}()

	otel.SetMeterProvider(provider)

	assert.NotNil(t, cacheCounter)

	ObserveHit(context.Background(), attribute.String("test", "test"))

	collectedMetrics := &metricdata.ResourceMetrics{}
	assert.NoError(t, read.Collect(context.Background(), collectedMetrics))

	assert.Equal(t, 1, len(collectedMetrics.ScopeMetrics))

	ObserveMiss(context.Background(), attribute.String("test", "test"))

	collectedMetrics = &metricdata.ResourceMetrics{}
	assert.NoError(t, read.Collect(context.Background(), collectedMetrics))

	assert.Equal(t, 1, len(collectedMetrics.ScopeMetrics))
}
