package mqtt

import (
	"context"
	"time"

	"github.com/beatlabs/patron/observability"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	packageName    = "mqtt"
	topicAttribute = "topic"
)

var durationHistogram metric.Int64Histogram

func init() {
	var err error
	durationHistogram, err = patronmetric.Int64Histogram(packageName, "mqtt.publish.duration", "MQTT publish duration.", "ms")
	if err != nil {
		panic(err)
	}
}

func topicAttr(topic string) attribute.KeyValue {
	return attribute.String(topicAttribute, topic)
}

func observePublish(ctx context.Context, start time.Time, topic string, err error) {
	durationHistogram.Record(ctx, time.Since(start).Milliseconds(),
		metric.WithAttributes(observability.ClientAttribute(packageName), topicAttr(topic),
			observability.StatusAttribute(err)))
}
