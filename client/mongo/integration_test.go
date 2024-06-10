//go:build integration

package mongo

import (
	"context"
	"testing"

	"github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.opentelemetry.io/otel"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TODO: introduce metrics?

func TestConnectAndExecute(t *testing.T) {
	// Setup tracing monitoring
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	// Metrics monitoring set up
	read := metricsdk.NewManualReader()
	provider := metricsdk.NewMeterProvider(metricsdk.WithReader(read))
	defer func() {
		assert.NoError(t, provider.Shutdown(context.Background()))
	}()

	otel.SetMeterProvider(provider)

	ctx := context.Background()

	client, err := Connect(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, client)

	t.Run("success", func(t *testing.T) {
		exp.Reset()
		err = client.Ping(ctx, nil)
		require.NoError(t, err)
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assert.Len(t, exp.GetSpans(), 1)
		// Metrics
		collectedMetrics := &metricdata.ResourceMetrics{}
		assert.NoError(t, read.Collect(context.Background(), collectedMetrics))
		assert.Equal(t, 1, len(collectedMetrics.ScopeMetrics))
	})

	t.Run("failure", func(t *testing.T) {
		exp.Reset()
		names, err := client.ListDatabaseNames(ctx, bson.M{})
		assert.Error(t, err)
		assert.Empty(t, names)
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assert.Len(t, exp.GetSpans(), 1)
		// Metrics
		collectedMetrics := &metricdata.ResourceMetrics{}
		assert.NoError(t, read.Collect(context.Background(), collectedMetrics))
		assert.Equal(t, 1, len(collectedMetrics.ScopeMetrics))
	})
}
