package kafka

import (
	"context"

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
	consumerErrorsGauge           metric.Int64Counter
	topicPartitionOffsetDiffGauge metric.Float64Gauge
	messageStatusCount            metric.Int64Counter
	messageStatusAttr             = attribute.String(statusAttribute, messageReceived)
	messageProcessedAttr          = attribute.String(statusAttribute, messageProcessed)
	messageErroredAttr            = attribute.String(statusAttribute, messageErrored)
	messageSkippedAttr            = attribute.String(statusAttribute, messageSkipped)
)

func init() {
	consumerErrorsGauge = patronmetric.Int64Counter(packageName, "kafka.consumer.errors", "Kafka consumer error counter.", "s")
	topicPartitionOffsetDiffGauge = patronmetric.Float64Gauge(packageName, "kafka.consumer.offset.diff", "Kafka topic partition diff gauge.", "1")
	messageStatusCount = patronmetric.Int64Counter(packageName, "kafka.message.status", "Kafka message status counter.", "1")
}

func consumerErrorsInc(ctx context.Context, name string) {
	consumerErrorsGauge.Add(ctx, 1, metric.WithAttributes(attribute.String("consumer", name)))
}

func topicPartitionOffsetDiffGaugeSet(ctx context.Context, group, topic string, partition int32, high, offset int64) {
	topicPartitionOffsetDiffGauge.Record(ctx, float64(high-offset), metric.WithAttributes(
		attribute.String("group", group),
		attribute.String("topic", topic),
		attribute.Int64("partition", int64(partition)),
	))
}

func messageStatusCountInc(ctx context.Context, status, group, topic string) {
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

	messageStatusCount.Add(ctx, 1, metric.WithAttributes(
		attribute.String("group", group),
		attribute.String("topic", topic),
		statusAttr,
	))
}
