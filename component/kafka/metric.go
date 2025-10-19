package kafka

import (
	"context"
	"sync"

	patronmetric "github.com/beatlabs/patron/observability/metric"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	packageName      = "kafka"
	messageReceived  = "received"
	messageProcessed = "processed"
	messageErrored   = "errored"
	messageSkipped   = "skipped"
	statusAttribute  = "status"
)

var (
	messageStatusAttr    = attribute.String(statusAttribute, messageReceived)
	messageProcessedAttr = attribute.String(statusAttribute, messageProcessed)
	messageErroredAttr   = attribute.String(statusAttribute, messageErrored)
	messageSkippedAttr   = attribute.String(statusAttribute, messageSkipped)
)

type kafkaMetrics struct {
	consumerErrorsGauge           metric.Int64Counter
	topicPartitionOffsetDiffGauge metric.Float64Gauge
	messageStatusCount            metric.Int64Counter
}

var (
	metricsInstance *kafkaMetrics
	metricsOnce     sync.Once
	errMetrics      error
)

func setupMetricsOnce() (*kafkaMetrics, error) {
	metricsOnce.Do(func() {
		consumerErrorsGauge, err := patronmetric.Int64Counter(packageName, "kafka.consumer.errors", "Kafka consumer error counter.", "s")
		if err != nil {
			errMetrics = err
			return
		}

		topicPartitionOffsetDiffGauge, err := patronmetric.Float64Gauge(packageName, "kafka.consumer.offset.diff", "Kafka topic partition diff gauge.", "1")
		if err != nil {
			errMetrics = err
			return
		}

		messageStatusCount, err := patronmetric.Int64Counter(packageName, "kafka.message.status", "Kafka message status counter.", "1")
		if err != nil {
			errMetrics = err
			return
		}

		metricsInstance = &kafkaMetrics{
			consumerErrorsGauge:           consumerErrorsGauge,
			topicPartitionOffsetDiffGauge: topicPartitionOffsetDiffGauge,
			messageStatusCount:            messageStatusCount,
		}
	})

	return metricsInstance, errMetrics
}

func consumerErrorsInc(ctx context.Context, name string) {
	m, _ := setupMetricsOnce()
	if m == nil {
		return
	}
	m.consumerErrorsGauge.Add(ctx, 1, metric.WithAttributes(attribute.String("consumer", name)))
}

func topicPartitionOffsetDiffGaugeSet(ctx context.Context, group, topic string, partition int32, high, offset int64) {
	m, _ := setupMetricsOnce()
	if m == nil {
		return
	}
	m.topicPartitionOffsetDiffGauge.Record(ctx, float64(high-offset), metric.WithAttributes(
		attribute.String("group", group),
		attribute.String("topic", topic),
		attribute.Int64("partition", int64(partition)),
	))
}

func messageStatusCountInc(ctx context.Context, status, group, topic string) {
	m, _ := setupMetricsOnce()
	if m == nil {
		return
	}
	var statusAttr attribute.KeyValue
	switch status {
	case messageProcessed:
		statusAttr = messageProcessedAttr
	case messageErrored:
		statusAttr = messageErroredAttr
	case messageSkipped:
		statusAttr = messageSkippedAttr
	default:
		statusAttr = messageStatusAttr
	}

	m.messageStatusCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("group", group),
		attribute.String("topic", topic),
		statusAttr,
	))
}
