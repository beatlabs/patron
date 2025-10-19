package es

import (
	"context"
	"net/http"
	"testing"

	"github.com/beatlabs/patron/internal/test"
)

// Unit tests for the instrumentation wrapper to ensure metrics are emitted.

func TestInstrumentation_SuccessEmitsMetric(t *testing.T) {
	ctx := context.Background()
	shutdown, collect := test.SetupMetrics(ctx, t)
	defer shutdown()

	instr, err := newMetricInstrumentation("1.0.0")
	if err != nil {
		t.Fatal(err)
	}

	// Start a request span/context
	ctx = instr.Start(ctx, "indices.create")

	// Simulate a response with 200 OK
	res := &http.Response{StatusCode: http.StatusOK, Header: http.Header{}}
	instr.AfterResponse(ctx, res)

	// Close should record a histogram sample
	instr.Close(ctx)

	rm := collect(1) // Expect 1 metric

	// Verify the metric exists
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricName { // es.request.duration
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected %s metric to be recorded", metricName)
	}
}

func TestInstrumentation_FailureEmitsMetric(t *testing.T) {
	ctx := context.Background()
	shutdown, collect := test.SetupMetrics(ctx, t)
	defer shutdown()

	instr, err := newMetricInstrumentation("1.0.0")
	if err != nil {
		t.Fatal(err)
	}

	// Start a request span/context
	ctx = instr.Start(ctx, "indices.create")

	// Simulate an error before response
	instr.RecordError(ctx, errStatusCode)

	// Close should record a histogram sample even on error
	instr.Close(ctx)

	rm := collect(1) // Expect 1 metric  // Don't expect any metrics initially
	// Verify the metric exists
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == metricName { // es.request.duration
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected %s metric to be recorded on failure", metricName)
	}
}
