package amqp

import (
	"context"
	"time"

	"github.com/beatlabs/patron/observability"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const packageName = "amqp"

var (
	messageAgeGauge       metric.Float64Gauge
	messageCounter        metric.Int64Counter
	messageQueueSizeGauge metric.Int64Gauge

	ackStateAttr     = attribute.String("state", string(ackMessageState))
	nackStateAttr    = attribute.String("state", string(nackMessageState))
	fetchedStateAttr = attribute.String("state", string(fetchedMessageState))
)

func init() {
	messageAgeGauge = patronmetric.Float64Gauge(packageName, "amqp.message.age", "AMQP message age.", "s")
	messageCounter = patronmetric.Int64Counter(packageName, "amqp.message.counter", "AMQP message counter.", "1")
	messageQueueSizeGauge = patronmetric.Int64Gauge(packageName, "amqp.queue.size", "AMQP message queue size.", "1")
}

func observeMessageCountInc(ctx context.Context, queue string, state messageState, err error) {
	var stateAttr attribute.KeyValue
	switch state {
	case ackMessageState:
		stateAttr = ackStateAttr
	case nackMessageState:
		stateAttr = nackStateAttr
	case fetchedMessageState:
		stateAttr = fetchedStateAttr
	}

	messageCounter.Add(ctx, 1, metric.WithAttributes(queueAttributes(queue), stateAttr,
		observability.StatusAttribute(err)))
}

func observeReceivedMessageStats(ctx context.Context, queue string, timestamp time.Time) {
	messageAgeGauge.Record(ctx, time.Now().UTC().Sub(timestamp).Seconds(),
		metric.WithAttributes(queueAttributes(queue)))
	observeMessageCountInc(ctx, queue, fetchedMessageState, nil)
}

func observeQueueSize(ctx context.Context, queue string, size int) {
	messageQueueSizeGauge.Record(ctx, int64(size), metric.WithAttributes(queueAttributes(queue)))
}

func queueAttributes(queue string) attribute.KeyValue {
	return attribute.String("queue", queue)
}
