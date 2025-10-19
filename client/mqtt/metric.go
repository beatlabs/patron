package mqtt

import (
	"context"
	"time"

	"github.com/beatlabs/patron/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	packageName    = "mqtt"
	topicAttribute = "topic"
)

func topicAttr(topic string) attribute.KeyValue {
	return attribute.String(topicAttribute, topic)
}

func (p *Publisher) observePublish(ctx context.Context, start time.Time, topic string, err error) {
	p.durationHistogram.Record(ctx, time.Since(start).Milliseconds(),
		metric.WithAttributes(observability.ClientAttribute(packageName), topicAttr(topic),
			observability.StatusAttribute(err)))
}
