// Package observability provides functionality for initializing OpenTelemetry's traces and metrics.
// It includes methods for setting up and shutting down the observability components.
package observability

import (
	"context"
	"log/slog"

	"github.com/beatlabs/patron/observability/log"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Provider represents the observability provider that includes the metric and trace providers.
type Provider struct {
	mp *metric.MeterProvider
	tp *trace.TracerProvider
}

var (
	// SucceededAttribute is the attribute key-value pair for a succeeded operation.
	SucceededAttribute = attribute.String("status", "succeeded")
	// FailedAttribute is the attribute key-value pair for a failed operation.
	FailedAttribute = attribute.String("status", "failed")
)

// ComponentAttribute returns the attribute key-value pair for a component.
func ComponentAttribute(name string) attribute.KeyValue {
	return attribute.String("component", name)
}

// ClientAttribute returns the attribute key-value pair for a client.
func ClientAttribute(name string) attribute.KeyValue {
	return attribute.String("client", name)
}

// StatusAttribute returns the attribute key-value pair for the status of an operation.
func StatusAttribute(err error) attribute.KeyValue {
	if err != nil {
		return FailedAttribute
	}
	return SucceededAttribute
}

// Config represents the configuration for setting up traces, metrics and logs.
type Config struct {
	Name      string
	Version   string
	LogConfig log.Config
}

// Setup initializes OpenTelemetry's traces and metrics.
// It creates a resource with the given name and version, sets up the metric and trace providers,
// and returns a Provider containing the initialized providers.
func Setup(ctx context.Context, cfg Config) (*Provider, error) {
	log.Setup(&cfg.LogConfig)

	res, err := createResource(cfg.Name, cfg.Version)
	if err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.TraceContext{})

	metricProvider, err := patronmetric.Setup(ctx, res)
	if err != nil {
		return nil, err
	}
	traceProvider, err := patrontrace.SetupGRPC(ctx, cfg.Name, res)
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
		slog.Error("failed to flush metrics", log.ErrorAttr(err))
	}
	err = p.mp.Shutdown(ctx)
	if err != nil {
		return err
	}

	err = p.tp.ForceFlush(ctx)
	if err != nil {
		slog.Error("failed to flush traces", log.ErrorAttr(err))
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
