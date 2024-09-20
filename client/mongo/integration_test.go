//go:build integration

package mongo

import (
	"context"
	"testing"

	"github.com/beatlabs/patron/internal/test"
	"github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestConnectAndExecute(t *testing.T) {
	// Setup tracing monitoring
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	shutdownProvider, read := test.SetupMetrics(t)
	defer shutdownProvider()

	ctx := context.Background()

	client, err := Connect(ctx)
	require.NoError(t, err)
	assert.NotNil(t, client)

	t.Run("success", func(t *testing.T) {
		exp.Reset()
		err = client.Ping(ctx, nil)
		require.NoError(t, err)
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		assert.Len(t, exp.GetSpans(), 1)
		// Metrics
		collectedMetrics := &metricdata.ResourceMetrics{}
		require.NoError(t, read.Collect(context.Background(), collectedMetrics))
		assert.Len(t, collectedMetrics.ScopeMetrics, 1)
		assert.Len(t, collectedMetrics.ScopeMetrics[0].Metrics, 1)
	})

	t.Run("failure", func(t *testing.T) {
		exp.Reset()
		names, err := client.ListDatabaseNames(ctx, bson.M{})
		require.Error(t, err)
		assert.Empty(t, names)
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		assert.Len(t, exp.GetSpans(), 1)
		// Metrics
		collectedMetrics := &metricdata.ResourceMetrics{}
		require.NoError(t, read.Collect(context.Background(), collectedMetrics))
		assert.Len(t, collectedMetrics.ScopeMetrics, 1)
		assert.Len(t, collectedMetrics.ScopeMetrics[0].Metrics, 1)
	})
}
