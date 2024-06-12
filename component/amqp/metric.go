package amqp

import (
	"context"
	"time"

	"github.com/beatlabs/patron/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	messageAgeGauge       metric.Float64Gauge
	messageCounter        metric.Int64Counter
	messageQueueSizeGauge metric.Int64Gauge

	ackStateAttr     = attribute.String("state", string(ackMessageState))
	nackStateAttr    = attribute.String("state", string(nackMessageState))
	fetchedStateAttr = attribute.String("state", string(fetchedMessageState))
)

func init() {
	var err error
	messageAgeGauge, err = otel.Meter("amqp").Float64Gauge("amqp.message.age",
		metric.WithDescription("AMQP message age."),
		metric.WithUnit("s"))
	if err != nil {
		panic(err)
	}

	messageCounter, err = otel.Meter("amqp").Int64Counter("amqp.message.counter",
		metric.WithDescription("AMQP message counter."),
		metric.WithUnit("1"))
	if err != nil {
		panic(err)
	}

	messageQueueSizeGauge, err = otel.Meter("amqp").Int64Gauge("amqp.queue.size",
		metric.WithDescription("AMQP message queue size."),
		metric.WithUnit("1"))
	if err != nil {
		panic(err)
	}
}

func observeMessageCountInc(ctx context.Context, queue string, state messageState, err error) {
	messageCounter.Add(ctx, 1, metric.WithAttributes(queueAttributes(queue),
		attribute.String("state", string(state)),
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
