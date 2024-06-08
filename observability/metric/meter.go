package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

var meter metric.Meter

func Meter() metric.Meter {
	return meter
}

// Setup initializes OpenTelemetry's metrics.
func Setup(ctx context.Context, name string, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	meterProvider, err := newMeterProvider(ctx, res)
	if err != nil {
		return nil, err
	}

	return SetupWithMeterProvider(ctx, name, meterProvider)
}

// SetupWithMeterProvider initializes OpenTelemetry's metrics with a custom meter provider.
func SetupWithMeterProvider(ctx context.Context, name string, provider *sdkmetric.MeterProvider) (*sdkmetric.MeterProvider, error) {
	otel.SetMeterProvider(provider)

	meter = otel.Meter(name)

	return provider, nil
}

func newMeterProvider(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	metricExporter, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(20*time.Second))),
	)
	return meterProvider, nil
}
