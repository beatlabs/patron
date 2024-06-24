package mqtt

import (
	"context"
	"time"

	"github.com/beatlabs/patron/observability"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const packageName = "mqtt"

var durationHistogram metric.Int64Histogram

func init() {
	durationHistogram = patronmetric.Int64Histogram(packageName, "mqtt.publish.duration", "MQTT publish duration.", "ms")
}

func topicAttr(topic string) attribute.KeyValue {
	return attribute.String("topic", topic)
}

func observePublish(ctx context.Context, start time.Time, topic string, err error) {
	durationHistogram.Record(ctx, time.Since(start).Milliseconds(),
		metric.WithAttributes(observability.ClientAttribute(packageName), topicAttr(topic),
			observability.StatusAttribute(err)))
}
