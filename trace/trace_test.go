package trace

import (
	"context"
	"testing"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

func TestSetup_Tracer_Close(t *testing.T) {
	err := Setup("TEST", "1.0.0", "0.0.0.0:6831", "const", 1)
	assert.NoError(t, err)
	err = Close()
	assert.NoError(t, err)
	Version = "dev"
}

func TestStartFinishConsumerSpan(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	hdr := map[string]string{"key": "val"}
	sp, ctx := ConsumerSpan(context.Background(), "123", "custom-consumer", "corID", hdr)
	assert.NotNil(t, sp)
	assert.NotNil(t, ctx)
	assert.IsType(t, &mocktracer.MockSpan{}, sp)
	jsp := sp.(*mocktracer.MockSpan)
	assert.NotNil(t, jsp)
	assert.Equal(t, "123", jsp.OperationName)
	SpanError(sp)
	assert.NotNil(t, sp)
	rawSpan := mtr.FinishedSpans()[0]
	assert.Equal(t, map[string]interface{}{
		"span.kind":     ext.SpanKindConsumerEnum,
		"component":     "custom-consumer",
		"error":         true,
		"version":       "dev",
		"correlationID": "corID",
	}, rawSpan.Tags())
}

func TestStartFinishChildSpan(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	tag := opentracing.Tag{Key: "key", Value: "value"}
	sp, ctx := ConsumerSpan(context.Background(), "123", "custom-consumer", "corID", nil, tag)
	assert.NotNil(t, sp)
	assert.NotNil(t, ctx)
	childSp, childCtx := ChildSpan(ctx, "123", "cmp", tag)
	assert.NotNil(t, childSp)
	assert.NotNil(t, childCtx)
	childSp.LogKV("log event")
	assert.IsType(t, &mocktracer.MockSpan{}, childSp)
	jsp := childSp.(*mocktracer.MockSpan)
	assert.NotNil(t, jsp)
	assert.Equal(t, "123", jsp.OperationName)
	SpanError(childSp)
	assert.NotNil(t, childSp)
	rawSpan := mtr.FinishedSpans()[0]
	assert.Equal(t, map[string]interface{}{
		"component": "cmp",
		"error":     true,
		"key":       "value",
		"version":   "dev",
	}, rawSpan.Tags())
	SpanSuccess(sp)
	rawSpan = mtr.FinishedSpans()[1]
	assert.Equal(t, map[string]interface{}{
		"component":     "custom-consumer",
		"error":         false,
		"version":       "dev",
		"key":           "value",
		"span.kind":     ext.SpanKindConsumerEnum,
		"correlationID": "corID",
	}, rawSpan.Tags())
}

func TestComponentOpName(t *testing.T) {
	assert.Equal(t, "cmp target", ComponentOpName("cmp", "target"))
}
