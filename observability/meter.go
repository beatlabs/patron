package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var meter metric.Meter

func setupMeter(name string) error {
	meter = otel.Meter(name)
	return nil
}

func Meter() metric.Meter {
	return meter
}
