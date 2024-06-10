package mqtt

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	componentAttr     = attribute.String("component", "mqtt")
	succeededAttr     = attribute.String("status", "succeeded")
	failedAttr        = attribute.String("status", "failed")
	durationHistogram metric.Int64Histogram
)

func init() {
	var err error
	durationHistogram, err = otel.Meter("mqtt").Int64Histogram("mqtt.publish.duration",
		metric.WithDescription("MQTT publish duration."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		panic(err)
	}
}

func topicAttr(topic string) attribute.KeyValue {
	return attribute.String("topic", topic)
}

func observePublish(ctx context.Context, start time.Time, topic string, err error) {
	var statusAttr attribute.KeyValue
	if err != nil {
		statusAttr = failedAttr
	} else {
		statusAttr = succeededAttr
	}

	durationHistogram.Record(ctx, time.Since(start).Milliseconds(),
		metric.WithAttributes(componentAttr, succeededAttr, topicAttr(topic), statusAttr))
}
