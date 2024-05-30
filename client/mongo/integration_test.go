//go:build integration
// +build integration

package mongo

import (
	"context"
	"testing"

	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// TODO: introduce metrics?

func TestConnectAndExecute(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tracePublisher, err := patrontrace.Setup("test", nil, exp)
	require.NoError(t, err)
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
	})

	t.Run("failure", func(t *testing.T) {
		exp.Reset()
		names, err := client.ListDatabaseNames(ctx, bson.M{})
		assert.Error(t, err)
		assert.Empty(t, names)
		assert.NoError(t, tracePublisher.ForceFlush(ctx))
		assert.Len(t, exp.GetSpans(), 1)
	})
}
