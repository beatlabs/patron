package mqtt

import (
	"context"
	"time"

	"github.com/beatlabs/patron/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var durationHistogram metric.Int64Histogram

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
	durationHistogram.Record(ctx, time.Since(start).Milliseconds(),
		metric.WithAttributes(observability.ClientAttribute("mqtt"), topicAttr(topic),
			observability.StatusAttribute(err)))
}
