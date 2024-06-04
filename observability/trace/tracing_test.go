package trace

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestSetupGRPC(t *testing.T) {
	got, err := SetupGRPC(context.Background(), "test", resource.Default())
	assert.NoError(t, err)
	assert.NotNil(t, got)
}

func TestComponentOpName(t *testing.T) {
	assert.Equal(t, "cmp target", ComponentOpName("cmp", "target"))
}

func TestSetSpanStatus(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := Setup("test", nil, exp)

	t.Run("Success", func(t *testing.T) {
		t.Cleanup(func() { exp.Reset() })
		ctx, sp := StartSpan(context.Background(), "test")
		assert.NotNil(t, ctx)
		SetSpanSuccess(sp)
		sp.End()
		assert.NoError(t, tracePublisher.ForceFlush(context.Background()))
		spans := exp.GetSpans()
		assert.Len(t, spans, 1)
		assert.Equal(t, "test", spans[0].Name)
		assert.Equal(t, codes.Ok, spans[0].Status.Code)
	})

	t.Run("Error", func(t *testing.T) {
		t.Cleanup(func() { exp.Reset() })
		ctx, sp := StartSpan(context.Background(), "test")
		assert.NotNil(t, ctx)
		SetSpanError(sp, "error msg", errors.New("error"))
		sp.End()
		assert.NoError(t, tracePublisher.ForceFlush(context.Background()))
		spans := exp.GetSpans()
		assert.Len(t, spans, 1)
		assert.Equal(t, "test", spans[0].Name)
		assert.Equal(t, codes.Error, spans[0].Status.Code)
		assert.Equal(t, "error msg", spans[0].Status.Description)
		assert.Len(t, spans[0].Events, 1)
		assert.Equal(t, "exception", spans[0].Events[0].Name)
		assert.Equal(t, "exception.type", string(spans[0].Events[0].Attributes[0].Key))
		assert.Equal(t, "*errors.errorString", spans[0].Events[0].Attributes[0].Value.AsString())
		assert.Equal(t, "exception.message", string(spans[0].Events[0].Attributes[1].Key))
		assert.Equal(t, "error", spans[0].Events[0].Attributes[1].Value.AsString())
	})
}
