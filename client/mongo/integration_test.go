//go:build integration
// +build integration

package mongo

import (
	"context"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestConnectAndExecute(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	defer mtr.Reset()
	client, err := Connect(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, client)

	t.Run("success", func(t *testing.T) {
		t.Cleanup(func() {
			mtr.Reset()
			cmdDurationMetrics.Reset()
		})
		err = client.Ping(context.Background(), nil)
		require.NoError(t, err)

		sp := mtr.FinishedSpans()[0]
		assert.Equal(t, "ping", sp.OperationName)
		assert.Equal(t, map[string]interface{}{
			"component": "mongo-client",
			"error":     false,
			"span.kind": ext.SpanKindEnum("client"),
			"version":   "dev",
		}, sp.Tags())

		assert.Equal(t, 1, testutil.CollectAndCount(cmdDurationMetrics, "client_mongo_cmd_duration_seconds"))
	})

	t.Run("failure", func(t *testing.T) {
		t.Cleanup(func() {
			mtr.Reset()
			cmdDurationMetrics.Reset()
		})
		names, err := client.ListDatabaseNames(context.Background(), bson.M{})
		assert.Error(t, err)
		assert.Empty(t, names)

		sp := mtr.FinishedSpans()[0]
		assert.Equal(t, "listDatabases", sp.OperationName)
		assert.Equal(t, map[string]interface{}{
			"component": "mongo-client",
			"error":     true,
			"span.kind": ext.SpanKindEnum("client"),
			"version":   "dev",
		}, sp.Tags())

		assert.Equal(t, 1, testutil.CollectAndCount(cmdDurationMetrics, "client_mongo_cmd_duration_seconds"))
	})
}
