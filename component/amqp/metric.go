package amqp

import (
	"context"
	"sync"
	"time"

	"github.com/beatlabs/patron/observability"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	packageName    = "amqp"
	stateAttribute = "state"
)

var (
	ackStateAttr     = attribute.String(stateAttribute, string(ackMessageState))
	nackStateAttr    = attribute.String(stateAttribute, string(nackMessageState))
	fetchedStateAttr = attribute.String(stateAttribute, string(fetchedMessageState))
)

type amqpMetrics struct {
	messageAgeGauge       metric.Float64Gauge
	messageCounter        metric.Int64Counter
	messageQueueSizeGauge metric.Int64Gauge
}

var (
	metricsInstance *amqpMetrics
	metricsOnce     sync.Once
	errMetrics      error
)

func setupMetricsOnce() (*amqpMetrics, error) {
	metricsOnce.Do(func() {
		messageAgeGauge, err := patronmetric.Float64Gauge(packageName, "amqp.message.age", "AMQP message age.", "s")
		if err != nil {
			errMetrics = err
			return
		}

		messageCounter, err := patronmetric.Int64Counter(packageName, "amqp.message.counter", "AMQP message counter.", "1")
		if err != nil {
			errMetrics = err
			return
		}

		messageQueueSizeGauge, err := patronmetric.Int64Gauge(packageName, "amqp.queue.size", "AMQP message queue size.", "1")
		if err != nil {
			errMetrics = err
			return
		}

		metricsInstance = &amqpMetrics{
			messageAgeGauge:       messageAgeGauge,
			messageCounter:        messageCounter,
			messageQueueSizeGauge: messageQueueSizeGauge,
		}
	})

	return metricsInstance, errMetrics
}

func observeMessageCountInc(ctx context.Context, queue string, state messageState, err error) {
	m, _ := setupMetricsOnce()
	if m == nil {
		return
	}

	var stateAttr attribute.KeyValue
	switch state {
	case ackMessageState:
		stateAttr = ackStateAttr
	case nackMessageState:
		stateAttr = nackStateAttr
	case fetchedMessageState:
		stateAttr = fetchedStateAttr
	}

	m.messageCounter.Add(ctx, 1, metric.WithAttributes(queueAttributes(queue), stateAttr,
		observability.StatusAttribute(err)))
}

func observeReceivedMessageStats(ctx context.Context, queue string, timestamp time.Time) {
	m, _ := setupMetricsOnce()
	if m == nil {
		return
	}

	m.messageAgeGauge.Record(ctx, time.Now().UTC().Sub(timestamp).Seconds(),
		metric.WithAttributes(queueAttributes(queue)))
	observeMessageCountInc(ctx, queue, fetchedMessageState, nil)
}

func observeQueueSize(ctx context.Context, queue string, size int) {
	m, _ := setupMetricsOnce()
	if m == nil {
		return
	}

	m.messageQueueSizeGauge.Record(ctx, int64(size), metric.WithAttributes(queueAttributes(queue)))
}

func queueAttributes(queue string) attribute.KeyValue {
	return attribute.String("queue", queue)
}
