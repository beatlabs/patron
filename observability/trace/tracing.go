package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

// Tracer returns the global tracer.
func Tracer() trace.Tracer {
	return tracer
}

// SetupGRPC configures the global tracer with the OTLP gRPC exporter.
func SetupGRPC(ctx context.Context, name string, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exp, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	return Setup(name, res, exp)
}

// Setup TraceProvider with the given resource and exporter.
func Setup(name string, res *resource.Resource, exp sdktrace.SpanExporter) (*sdktrace.TracerProvider, error) {
	tp := newTraceProvider(res, exp)

	otel.SetTracerProvider(tp)
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)

	tracer = tp.Tracer(name)

	return tp, nil
}

func newTraceProvider(res *resource.Resource, exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
}
