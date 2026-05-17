package metric

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/resource"
)

var errInstrumentCreation = errors.New("instrument creation failed")

type failingMeterProvider struct{ noop.MeterProvider }

func (f failingMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return failingMeter{}
}

type failingMeter struct{ noop.Meter }

func (f failingMeter) Float64Histogram(string, ...metric.Float64HistogramOption) (metric.Float64Histogram, error) {
	return panicFloat64Histogram{}, errInstrumentCreation
}

func (f failingMeter) Int64Histogram(string, ...metric.Int64HistogramOption) (metric.Int64Histogram, error) {
	return panicInt64Histogram{}, errInstrumentCreation
}

func (f failingMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return panicInt64Counter{}, errInstrumentCreation
}

func (f failingMeter) Float64Gauge(string, ...metric.Float64GaugeOption) (metric.Float64Gauge, error) {
	return panicFloat64Gauge{}, errInstrumentCreation
}

func (f failingMeter) Int64Gauge(string, ...metric.Int64GaugeOption) (metric.Int64Gauge, error) {
	return panicInt64Gauge{}, errInstrumentCreation
}

type panicFloat64Histogram struct{ noop.Float64Histogram }

func (p panicFloat64Histogram) Record(context.Context, float64, ...metric.RecordOption) {
	panic("unexpected float64 histogram use")
}

type panicInt64Histogram struct{ noop.Int64Histogram }

func (p panicInt64Histogram) Record(context.Context, int64, ...metric.RecordOption) {
	panic("unexpected int64 histogram use")
}

type panicInt64Counter struct{ noop.Int64Counter }

func (p panicInt64Counter) Add(context.Context, int64, ...metric.AddOption) {
	panic("unexpected int64 counter use")
}

type panicFloat64Gauge struct{ noop.Float64Gauge }

func (p panicFloat64Gauge) Record(context.Context, float64, ...metric.RecordOption) {
	panic("unexpected float64 gauge use")
}

type panicInt64Gauge struct{ noop.Int64Gauge }

func (p panicInt64Gauge) Record(context.Context, int64, ...metric.RecordOption) {
	panic("unexpected int64 gauge use")
}

func TestSetupWithMeterProvider(t *testing.T) {
	// Create a noop meter provider for testing
	provider := noop.NewMeterProvider()

	// Setup with the meter provider
	SetupWithMeterProvider(provider)

	// Verify the provider was set
	assert.Equal(t, provider, otel.GetMeterProvider())
}

func TestNewMeterProvider(t *testing.T) {
	t.Run("creates meter provider with default resource", func(t *testing.T) {
		t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
		t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")

		ctx := context.Background()
		res := resource.Default()

		mp, err := newMeterProvider(ctx, res)

		// In unit test environment without OTLP collector, this may fail
		// but we test that the function handles errors appropriately
		if err != nil {
			require.Error(t, err)
			assert.Nil(t, mp)
		} else {
			assert.NotNil(t, mp)
			// Clean up
			_ = mp.Shutdown(ctx)
		}
	})
}

func TestFloat64Histogram(t *testing.T) {
	// Setup a noop meter provider to avoid panics
	SetupWithMeterProvider(noop.NewMeterProvider())

	histogram := Float64Histogram("test-pkg", "test.histogram", "Test histogram", "ms")

	assert.NotNil(t, histogram)

	// Verify we can record values without panic
	ctx := context.Background()
	histogram.Record(ctx, 100.5)
	histogram.Record(ctx, 200.0)
}

func TestInt64Histogram(t *testing.T) {
	// Setup a noop meter provider to avoid panics
	SetupWithMeterProvider(noop.NewMeterProvider())

	histogram := Int64Histogram("test-pkg", "test.int.histogram", "Test int histogram", "ms")

	assert.NotNil(t, histogram)

	// Verify we can record values without panic
	ctx := context.Background()
	histogram.Record(ctx, 100)
	histogram.Record(ctx, 200)
}

func TestInt64Counter(t *testing.T) {
	// Setup a noop meter provider to avoid panics
	SetupWithMeterProvider(noop.NewMeterProvider())

	counter := Int64Counter("test-pkg", "test.counter", "Test counter", "1")

	assert.NotNil(t, counter)

	// Verify we can add values without panic
	ctx := context.Background()
	counter.Add(ctx, 1)
	counter.Add(ctx, 5)
}

func TestFloat64Gauge(t *testing.T) {
	// Setup a noop meter provider to avoid panics
	SetupWithMeterProvider(noop.NewMeterProvider())

	gauge := Float64Gauge("test-pkg", "test.gauge", "Test gauge", "celsius")

	assert.NotNil(t, gauge)

	// Verify we can record values without panic
	ctx := context.Background()
	gauge.Record(ctx, 25.5)
	gauge.Record(ctx, 30.0)
}

func TestInt64Gauge(t *testing.T) {
	// Setup a noop meter provider to avoid panics
	SetupWithMeterProvider(noop.NewMeterProvider())

	gauge := Int64Gauge("test-pkg", "test.int.gauge", "Test int gauge", "count")

	assert.NotNil(t, gauge)

	// Verify we can record values without panic
	ctx := context.Background()
	gauge.Record(ctx, 42)
	gauge.Record(ctx, 100)
}

func TestMetricFunctionsWithAttributes(t *testing.T) {
	// Setup a noop meter provider
	SetupWithMeterProvider(noop.NewMeterProvider())

	ctx := context.Background()

	t.Run("histogram with attributes", func(_ *testing.T) {
		histogram := Float64Histogram("test", "test.histogram.attrs", "Test", "ms")
		histogram.Record(ctx, 123.45, metric.WithAttributes())
	})

	t.Run("counter with attributes", func(_ *testing.T) {
		counter := Int64Counter("test", "test.counter.attrs", "Test", "1")
		counter.Add(ctx, 10, metric.WithAttributes())
	})

	t.Run("gauge with attributes", func(_ *testing.T) {
		gauge := Float64Gauge("test", "test.gauge.attrs", "Test", "units")
		gauge.Record(ctx, 50.0, metric.WithAttributes())
	})
}

func TestMetricCreationWithEmptyValues(t *testing.T) {
	// Setup a noop meter provider
	SetupWithMeterProvider(noop.NewMeterProvider())

	t.Run("empty package name", func(t *testing.T) {
		histogram := Float64Histogram("", "test.metric", "description", "unit")
		assert.NotNil(t, histogram)
	})

	t.Run("empty metric name", func(t *testing.T) {
		histogram := Float64Histogram("pkg", "", "description", "unit")
		assert.NotNil(t, histogram)
	})

	t.Run("empty description", func(t *testing.T) {
		histogram := Float64Histogram("pkg", "metric", "", "unit")
		assert.NotNil(t, histogram)
	})

	t.Run("empty unit", func(t *testing.T) {
		histogram := Float64Histogram("pkg", "metric", "description", "")
		assert.NotNil(t, histogram)
	})
}

func TestMultipleMetersFromSamePackage(t *testing.T) {
	// Setup a noop meter provider
	SetupWithMeterProvider(noop.NewMeterProvider())

	pkg := "test-package"

	// Create multiple metrics from the same package
	counter1 := Int64Counter(pkg, "counter1", "First counter", "1")
	counter2 := Int64Counter(pkg, "counter2", "Second counter", "1")
	histogram := Float64Histogram(pkg, "histogram", "Histogram", "ms")
	gauge := Int64Gauge(pkg, "gauge", "Gauge", "items")

	assert.NotNil(t, counter1)
	assert.NotNil(t, counter2)
	assert.NotNil(t, histogram)
	assert.NotNil(t, gauge)

	// Verify they all work
	ctx := context.Background()
	counter1.Add(ctx, 1)
	counter2.Add(ctx, 2)
	histogram.Record(ctx, 100.0)
	gauge.Record(ctx, 50)
}

func TestMetricFactoriesFallbackToNoopOnInstrumentCreationError(t *testing.T) {
	originalProvider := otel.GetMeterProvider()
	SetupWithMeterProvider(failingMeterProvider{})
	t.Cleanup(func() {
		SetupWithMeterProvider(originalProvider)
	})

	ctx := context.Background()

	assert.NotPanics(t, func() {
		Float64Histogram("test-pkg", "test.histogram", "Test histogram", "ms").Record(ctx, 1.5)
	})
	assert.NotPanics(t, func() {
		Int64Histogram("test-pkg", "test.int.histogram", "Test int histogram", "ms").Record(ctx, 1)
	})
	assert.NotPanics(t, func() {
		Int64Counter("test-pkg", "test.counter", "Test counter", "1").Add(ctx, 1)
	})
	assert.NotPanics(t, func() {
		Float64Gauge("test-pkg", "test.gauge", "Test gauge", "celsius").Record(ctx, 2.5)
	})
	assert.NotPanics(t, func() {
		Int64Gauge("test-pkg", "test.int.gauge", "Test int gauge", "count").Record(ctx, 2)
	})
}

func TestSetup_WithCustomResource(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")

	ctx := context.Background()

	// Create a custom resource
	res, err := resource.New(ctx,
		resource.WithAttributes(),
	)
	require.NoError(t, err)

	// Try to setup (may fail in test environment without OTLP collector)
	mp, err := Setup(ctx, res)

	if err != nil {
		// Expected in test environment without OTLP collector
		require.Error(t, err)
		assert.Nil(t, mp)
	} else {
		assert.NotNil(t, mp)
		assert.NotNil(t, otel.GetMeterProvider())
		// Clean up
		_ = mp.Shutdown(ctx)
	}
}
