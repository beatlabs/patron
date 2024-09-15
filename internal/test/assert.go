package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
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
