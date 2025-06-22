// Package trace provides observability over tracing.
package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

// StartSpan starts a span with the given name and context.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return tracer.Start(ctx, name, opts...) //nolint:spancheck
}

// SetupGRPC configures the global tracer with the OTLP gRPC exporter.
func SetupGRPC(ctx context.Context, name string, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exp, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	return Setup(name, res, exp), nil
}

// Setup TraceProvider with the given resource and exporter.
func Setup(name string, res *resource.Resource, exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	tp := newTraceProvider(res, exp)

	otel.SetTracerProvider(tp)
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)

	tracer = tp.Tracer(name)

	return tp
}

func newTraceProvider(res *resource.Resource, exp sdktrace.SpanExporter) *sdktrace.TracerProvider {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	}
	return sdktrace.NewTracerProvider(opts...)
}

// ComponentOpName returns an operation name for a component.
func ComponentOpName(cmp, target string) string {
	return cmp + " " + target
}

// SetSpanError sets the error status on the span.
func SetSpanError(span trace.Span, msg string, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, msg)
}

// SetSpanSuccess sets the success status on the span.
func SetSpanSuccess(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}
