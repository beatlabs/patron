//go:build integration

package redis

import (
	"context"
	"testing"

	"github.com/beatlabs/patron/observability/trace"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const (
	dsn = "localhost:6379"
)

func TestClient(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tracePublisher, err := trace.Setup("test", nil, exp)
	require.NoError(t, err)

	ctx, _ := trace.Tracer().Start(context.Background(), "test")

	cl, err := New(&redis.Options{Addr: dsn})
	assert.NoError(t, err)
	cmd := cl.Set(ctx, "key", "value", 0)
	res, err := cmd.Result()
	assert.NoError(t, err)
	assert.Equal(t, res, "OK")

	assert.NoError(t, tracePublisher.ForceFlush(ctx))

	assert.Len(t, exp.GetSpans(), 2)
}
