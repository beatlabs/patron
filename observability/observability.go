// package observability is based on OpenTelemetry.
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

type Provider struct {
	mp *metric.MeterProvider
	tp *trace.TracerProvider
}

// Shutdown flushes and shuts down the metrics and traces.
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

// Setup initializes OpenTelemetry's traces and metrics.
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

func createResource(name, version string) (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(name),
			semconv.ServiceVersion(version),
		))
}
