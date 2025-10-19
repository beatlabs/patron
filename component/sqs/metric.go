package sqs

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beatlabs/patron/observability"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	packageName    = "sqs"
	stateAttribute = "state"
)

var (
	ackStateAttr     = attribute.String(stateAttribute, string(ackMessageState))
	nackStateAttr    = attribute.String(stateAttribute, string(nackMessageState))
	fetchedStateAttr = attribute.String(stateAttribute, string(fetchedMessageState))
)

type sqsMetrics struct {
	messageAgeGauge       metric.Float64Gauge
	messageCounter        metric.Int64Counter
	messageQueueSizeGauge metric.Float64Gauge
}

var (
	metricsInstance *sqsMetrics
	metricsOnce     sync.Once
	errMetrics      error
)

func setupMetricsOnce() (*sqsMetrics, error) {
	metricsOnce.Do(func() {
		messageAgeGauge, err := patronmetric.Float64Gauge(packageName, "sqs.message.age", "SQS message age.", "s")
		if err != nil {
			errMetrics = err
			return
		}

		messageCounter, err := patronmetric.Int64Counter(packageName, "sqs.message.counter", "SQS message counter.", "1")
		if err != nil {
			errMetrics = err
			return
		}

		messageQueueSizeGauge, err := patronmetric.Float64Gauge(packageName, "sqs.queue.size", "SQS message queue size.", "1")
		if err != nil {
			errMetrics = err
			return
		}

		metricsInstance = &sqsMetrics{
			messageAgeGauge:       messageAgeGauge,
			messageCounter:        messageCounter,
			messageQueueSizeGauge: messageQueueSizeGauge,
		}
	})

	return metricsInstance, errMetrics
}

func observerMessageAge(ctx context.Context, queue string, attrs map[string]string) {
	m, _ := setupMetricsOnce()
	if m == nil {
		return
	}

	attr, ok := attrs[sqsAttributeSentTimestamp]
	if !ok || len(strings.TrimSpace(attr)) == 0 {
		return
	}
	timestamp, err := strconv.ParseInt(attr, 10, 64)
	if err != nil {
		return
	}
	m.messageAgeGauge.Record(ctx, time.Now().UTC().Sub(time.Unix(timestamp, 0)).Seconds(),
		metric.WithAttributes(queueAttributes(queue)))
}

func observeMessageCount(ctx context.Context, queue string, state messageState, err error, count int) {
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

	m.messageCounter.Add(ctx, int64(count), metric.WithAttributes(queueAttributes(queue), stateAttr,
		observability.StatusAttribute(err)))
}

func observeQueueSize(ctx context.Context, queue, state string, size float64) {
	m, _ := setupMetricsOnce()
	if m == nil {
		return
	}

	m.messageQueueSizeGauge.Record(ctx, size,
		metric.WithAttributes(queueAttributes(queue), attribute.String("state", state)))
}

func queueAttributes(queue string) attribute.KeyValue {
	return attribute.String("queue", queue)
}
