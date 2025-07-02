// Package metric provides observability over metrics.
package metric

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

// Setup initializes OpenTelemetry's metrics.
func Setup(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	meterProvider, err := newMeterProvider(ctx, res)
	if err != nil {
		return nil, err
	}

	SetupWithMeterProvider(meterProvider)

	return meterProvider, nil
}

// SetupWithMeterProvider initializes OpenTelemetry's metrics with a custom meter provider.
func SetupWithMeterProvider(provider metric.MeterProvider) {
	otel.SetMeterProvider(provider)
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

// Float64Histogram creates a float64 histogram metric.
func Float64Histogram(pkg, name, description, unit string) metric.Float64Histogram {
	histogram, err := otel.Meter(pkg).Float64Histogram(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		panic(err)
	}

	return histogram
}

// Int64Histogram creates an int64 histogram metric.
func Int64Histogram(pkg, name, description, unit string) metric.Int64Histogram {
	histogram, err := otel.Meter(pkg).Int64Histogram(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		panic(err)
	}

	return histogram
}

// Int64Counter creates an int64 counter metric.
func Int64Counter(pkg, name, description, unit string) metric.Int64Counter {
	counter, err := otel.Meter(pkg).Int64Counter(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		panic(err)
	}

	return counter
}

// Float64Gauge creates a float64 gauge metric.
func Float64Gauge(pkg, name, description, unit string) metric.Float64Gauge {
	gauge, err := otel.Meter(pkg).Float64Gauge(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		panic(err)
	}

	return gauge
}

// Int64Gauge creates an int64 gauge metric.
func Int64Gauge(pkg, name, description, unit string) metric.Int64Gauge {
	gauge, err := otel.Meter(pkg).Int64Gauge(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		panic(err)
	}

	return gauge
}
