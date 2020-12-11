package redis

import (
	"context"
	"testing"

	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

func TestSpan(t *testing.T) {
	opts := Options{Addr: "localhost"}
	c := New(opts)
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	tag := opentracing.Tag{Key: "key", Value: "value"}
	sp, req := c.startSpan(context.Background(), "localhost", "flushdb", tag)
	assert.NotNil(t, sp)
	assert.NotNil(t, req)
	assert.IsType(t, &mocktracer.MockSpan{}, sp)
	jsp := sp.(*mocktracer.MockSpan)
	assert.NotNil(t, jsp)
	trace.SpanSuccess(sp)
	rawSpan := mtr.FinishedSpans()[0]
	assert.Equal(t, map[string]interface{}{
		"component":    component,
		"db.instance":  "localhost",
		"db.statement": "flushdb",
		"db.type":      dbType,
		"error":        false,
		"key":          "value",
	}, rawSpan.Tags())
}
