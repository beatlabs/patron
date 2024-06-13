package sqs

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/beatlabs/patron/observability"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	messageAgeGauge       metric.Float64Gauge
	messageCounter        metric.Int64Counter
	messageQueueSizeGauge metric.Float64Gauge

	ackStateAttr     = attribute.String("state", string(ackMessageState))
	nackStateAttr    = attribute.String("state", string(nackMessageState))
	fetchedStateAttr = attribute.String("state", string(fetchedMessageState))
)

func init() {
	var err error
	messageAgeGauge, err = otel.Meter("sqs").Float64Gauge("sqs.message.age",
		metric.WithDescription("SQS message age."),
		metric.WithUnit("s"))
	if err != nil {
		panic(err)
	}

	messageCounter, err = otel.Meter("sqs").Int64Counter("sqs.message.counter",
		metric.WithDescription("AMQP message counter."),
		metric.WithUnit("1"))
	if err != nil {
		panic(err)
	}

	messageQueueSizeGauge, err = otel.Meter("sqs").Float64Gauge("sqs.queue.size",
		metric.WithDescription("SQS message queue size."),
		metric.WithUnit("1"))
	if err != nil {
		panic(err)
	}
}

func observerMessageAge(ctx context.Context, queue string, attributes map[string]string) {
	attribute, ok := attributes[sqsAttributeSentTimestamp]
	if !ok || len(strings.TrimSpace(attribute)) == 0 {
		return
	}
	timestamp, err := strconv.ParseInt(attribute, 10, 64)
	if err != nil {
		return
	}
	messageAgeGauge.Record(ctx, time.Now().UTC().Sub(time.Unix(timestamp, 0)).Seconds(),
		metric.WithAttributes(queueAttributes(queue)))
}

func observeMessageCount(ctx context.Context, queue string, state messageState, err error, count int) {
	var stateAttr attribute.KeyValue
	switch state {
	case ackMessageState:
		stateAttr = ackStateAttr
	case nackMessageState:
		stateAttr = nackStateAttr
	case fetchedMessageState:
		stateAttr = fetchedStateAttr
	}

	messageCounter.Add(ctx, int64(count), metric.WithAttributes(queueAttributes(queue), stateAttr,
		observability.StatusAttribute(err)))
}

func observeQueueSize(ctx context.Context, queue, state string, size float64) {
	messageQueueSizeGauge.Record(ctx, size,
		metric.WithAttributes(queueAttributes(queue), attribute.String("state", state)))
}

func queueAttributes(queue string) attribute.KeyValue {
	return attribute.String("queue", queue)
}
