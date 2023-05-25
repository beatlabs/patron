// Package trace provides trace support and helper methods.
package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	hostsTag   = "hosts"
	versionTag = "version"
	// TraceID is a label name for a request trace ID.
	TraceID = "traceID"
)

var (
	version    string
	tracerName string
	hostname   string
)

// Setup tracing.
func Setup(name, ver, host string) {
	tracerName = name
	version = ver
	hostname = host
}

func Tracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

// SpanComplete finishes a span with or without an error indicator.
func SpanComplete(sp trace.Span, err error) {
	if err != nil {
		sp.RecordError(err)
		sp.SetStatus(codes.Error, err.Error())
	}
	sp.End()
}

// ComponentOpName returns an operation name for a component.
func ComponentOpName(cmp, target string) string {
	return cmp + " " + target
}

// ProducerSpan returns a new span for a producer.
func ProducerSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, sp := otel.Tracer(tracerName).Start(ctx, name, trace.WithSpanKind(trace.SpanKindProducer))
	attrs = append(attrs, attribute.String(hostsTag, hostname))
	attrs = append(attrs, attribute.String(versionTag, version))
	sp.SetAttributes(attrs...)
	return ctx, sp
}
