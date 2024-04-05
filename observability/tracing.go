package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

// Tracer returns the global tracer.
func Tracer() trace.Tracer {
	return tracer
}

func setupTracing(ctx context.Context, name string, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	// TODO: setup options
	exp, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	// Create a new tracer provider with a batch span processor and the given exporter.
	tp, err := newTraceProvider(res, exp)
	if err != nil {
		return nil, err
	}

	otel.SetTracerProvider(tp)

	tracer = tp.Tracer(name)

	return tp, nil
}

func newTraceProvider(res *resource.Resource, exp sdktrace.SpanExporter) (*sdktrace.TracerProvider, error) {
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	), nil
}
