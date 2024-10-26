package sqs

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/beatlabs/patron/observability"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const packageName = "sqs"

var (
	messageAgeGauge       metric.Float64Gauge
	messageCounter        metric.Int64Counter
	messageQueueSizeGauge metric.Float64Gauge

	ackStateAttr     = attribute.String("state", string(ackMessageState))
	nackStateAttr    = attribute.String("state", string(nackMessageState))
	fetchedStateAttr = attribute.String("state", string(fetchedMessageState))
)

func init() {
	messageAgeGauge = patronmetric.Float64Gauge(packageName, "sqs.message.age", "SQS message age.", "s")
	messageCounter = patronmetric.Int64Counter(packageName, "sqs.message.counter", "SQS message counter.", "1")
	messageQueueSizeGauge = patronmetric.Float64Gauge(packageName, "sqs.queue.size", "SQS message queue size.", "1")
}

func observerMessageAge(ctx context.Context, queue string, attrs map[string]string) {
	attribute, ok := attrs[sqsAttributeSentTimestamp]
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
