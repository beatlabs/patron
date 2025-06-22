// Package test provides test helpers.
package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// AssertMetric asserts that a metric with the given name is present in the list of metrics.
func AssertMetric(t *testing.T, metrics []metricdata.Metrics, expectedName string) {
	for _, metric := range metrics {
		if metric.Name == expectedName {
			assert.Equal(t, expectedName, metric.Name)
			return
		}
	}

	assert.Fail(t, "metric not found", "metric %s not found", expectedName)
}

// AssertSpan asserts that the expected span is equal to the got span.
func AssertSpan(t *testing.T, expected tracetest.SpanStub, got tracetest.SpanStub) {
	assert.Equal(t, expected.Name, got.Name)
	assert.Equal(t, expected.SpanKind, got.SpanKind)
	assert.Equal(t, expected.Status, got.Status)
}
