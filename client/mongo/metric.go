package mongo

import (
	"context"

	"github.com/beatlabs/patron/observability"
	"go.mongodb.org/mongo-driver/event"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var durationHistogram metric.Int64Histogram

func init() {
	var err error
	durationHistogram, err = otel.Meter("mongo").Int64Histogram("mongo.duration",
		metric.WithDescription("Mongo command duration."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		panic(err)
	}
}

type observabilityMonitor struct {
	traceMonitor *event.CommandMonitor
}

func newObservabilityMonitor(traceMonitor *event.CommandMonitor) *event.CommandMonitor {
	m := &observabilityMonitor{
		traceMonitor: traceMonitor,
	}
	return &event.CommandMonitor{
		Started:   m.Started,
		Succeeded: m.Succeeded,
		Failed:    m.Failed,
	}
}

func (m *observabilityMonitor) Started(ctx context.Context, evt *event.CommandStartedEvent) {
	m.traceMonitor.Started(ctx, evt)
}

func (m *observabilityMonitor) Succeeded(ctx context.Context, evt *event.CommandSucceededEvent) {
	durationHistogram.Record(ctx, evt.Duration.Milliseconds(),
		metric.WithAttributes(observability.ClientAttribute("mongo"), observability.SucceededAttribute,
			commandAttr(evt.CommandName)))
	m.traceMonitor.Succeeded(ctx, evt)
}

func (m *observabilityMonitor) Failed(ctx context.Context, evt *event.CommandFailedEvent) {
	durationHistogram.Record(ctx, evt.Duration.Milliseconds(),
		metric.WithAttributes(observability.ClientAttribute("mongo"), observability.FailedAttribute,
			commandAttr(evt.CommandName)))
	m.traceMonitor.Failed(ctx, evt)
}

func commandAttr(cmdName string) attribute.KeyValue {
	return attribute.String("command", cmdName)
}
