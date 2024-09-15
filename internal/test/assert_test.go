package test

import (
	"testing"

	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestAssertMetric(t *testing.T) {
	metrics := []metricdata.Metrics{
		{Name: "metric1"},
		{Name: "metric2"},
	}
	AssertMetric(t, metrics, "metric1")
}
