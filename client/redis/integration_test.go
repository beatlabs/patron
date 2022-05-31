//go:build integration
// +build integration

package redis

import (
	"context"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	dsn = "localhost:6379"
)

func TestClient(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	defer mtr.Reset()

	cl := New(Options{
		Addr:     dsn,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	cmd := cl.Set(context.Background(), "key", "value", 0)
	res, err := cmd.Result()
	assert.NoError(t, err)
	assert.Equal(t, res, "OK")
	assert.Len(t, mtr.FinishedSpans(), 1)
	assert.Equal(t, mtr.FinishedSpans()[0].Tags()["component"], "redis")
	assert.Equal(t, mtr.FinishedSpans()[0].Tags()["error"], false)
	assert.Regexp(t, `:\d+`, mtr.FinishedSpans()[0].Tags()["db.instance"])
	assert.Equal(t, mtr.FinishedSpans()[0].Tags()["db.statement"], "set")
	assert.Equal(t, mtr.FinishedSpans()[0].Tags()["db.type"], "kv")
	// Metrics
	assert.Equal(t, 1, testutil.CollectAndCount(cmdDurationMetrics, "client_redis_cmd_duration_seconds"))
}
