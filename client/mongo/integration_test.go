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
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestConnectAndExecute(t *testing.T) {
	// Setup tracing monitoring
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	ctx := context.Background()

	shutdownProvider, collectAssertMetrics := test.SetupMetrics(ctx, t)
	defer shutdownProvider()

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
		_ = collectAssertMetrics(1)
	})

	t.Run("failure", func(t *testing.T) {
		exp.Reset()
		names, err := client.ListDatabaseNames(ctx, bson.M{})
		require.Error(t, err)
		assert.Empty(t, names)
		require.NoError(t, tracePublisher.ForceFlush(ctx))
		assert.Len(t, exp.GetSpans(), 1)
		// Metrics
		_ = collectAssertMetrics(1)
	})
}
