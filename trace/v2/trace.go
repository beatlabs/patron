// Package trace provides trace support and helper methods.
package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	// HostsTag is used to tag the component's hosts.
	HostsTag = "hosts"
	// VersionTag is used to tag the component's version.
	VersionTag = "version"
	// TraceID is a label name for a request trace ID.
	TraceID = "traceID"
)

// Version will be used to tag all traced components.
// It can be used to distinguish between dev, stage, and prod environments.
var Version = "dev"

func Tracer() trace.Tracer {
	return otel.Tracer("") // TODO: introduce name.
}

// SpanComplete finishes a span with or without an error indicator.
func SpanComplete(sp trace.Span, err error) {
	sp.RecordError(err)
	sp.End()
}

// ComponentOpName returns an operation name for a component.
func ComponentOpName(cmp, target string) string {
	return cmp + " " + target
}

func StartProducerSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, sp := otel.Tracer("").Start(ctx, name, trace.WithSpanKind(trace.SpanKindProducer))
	sp.SetAttributes(attrs...)
	return ctx, sp
}
