package test

import (
	"testing"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestAssertMetric(t *testing.T) {
	metrics := []metricdata.Metrics{
		{Name: "metric1"},
		{Name: "metric2"},
	}
	AssertMetric(t, metrics, "metric1")
}

func TestAssertSpan(t *testing.T) {
	expected := tracetest.SpanStub{
		Name:     "amqp queueName",
		SpanKind: trace.SpanKindConsumer,
		Status: tracesdk.Status{
			Code: codes.Ok,
		},
	}

	got := tracetest.SpanStub{
		Name:     "amqp queueName",
		SpanKind: trace.SpanKindConsumer,
		Status: tracesdk.Status{
			Code: codes.Ok,
		},
	}

	AssertSpan(t, expected, got)
}
