// Package observability provides functionality for initializing OpenTelemetry's traces and metrics.
// It includes methods for setting up and shutting down the observability components.
package observability

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
)

// Provider represents the observability provider that includes the metric and trace providers.
type Provider struct {
	mp *metric.MeterProvider
	tp *trace.TracerProvider
}

// Setup initializes OpenTelemetry's traces and metrics.
// It creates a resource with the given name and version, sets up the metric and trace providers,
// and returns a Provider containing the initialized providers.
func Setup(ctx context.Context, name, version string, conn *grpc.ClientConn) (*Provider, error) {
	res, err := createResource(name, version)
	if err != nil {
		return nil, err
	}

	metricProvider, err := setupMeter(ctx, name, res, conn)
	if err != nil {
		return nil, err
	}
	traceProvider, err := setupTracing(ctx, name, res, conn)
	if err != nil {
		return nil, err
	}
	return &Provider{
		mp: metricProvider,
		tp: traceProvider,
	}, nil
}

// Shutdown flushes and shuts down the metrics and traces.
// It forces a flush of metrics and traces, logs any errors encountered during flushing,
// and shuts down the metric and trace providers.
func (p *Provider) Shutdown(ctx context.Context) error {
	err := p.mp.ForceFlush(ctx)
	if err != nil {
		slog.Error("failed to flush metrics", slog.Any("error", err))
	}
	err = p.mp.Shutdown(ctx)
	if err != nil {
		return err
	}

	err = p.tp.ForceFlush(ctx)
	if err != nil {
		slog.Error("failed to flush traces", slog.Any("error", err))
	}

	return p.tp.Shutdown(ctx)
}

func createResource(name, version string) (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(name),
			semconv.ServiceVersion(version),
		))
}
