package kafka

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	messageReceived  = "received"
	messageProcessed = "processed"
	messageErrored   = "errored"
	messageSkipped   = "skipped"
)

var (
	consumerErrorsGauge           metric.Int64Counter
	topicPartitionOffsetDiffGauge metric.Float64Gauge
	messageStatusCount            metric.Int64Counter
	messageStatusAttr             = attribute.String("status", messageReceived)
	messageProcessedAttr          = attribute.String("status", messageProcessed)
	messageErroredAttr            = attribute.String("status", messageErrored)
	messageSkippedAttr            = attribute.String("status", messageSkipped)
)

func init() {
	var err error
	consumerErrorsGauge, err = otel.Meter("kafka").Int64Counter("kafka.consumer.errors",
		metric.WithDescription("Kafka consumer error counter."),
		metric.WithUnit("s"))
	if err != nil {
		panic(err)
	}

	topicPartitionOffsetDiffGauge, err = otel.Meter("kafka").Float64Gauge("kafka.consumer.offset.diff",
		metric.WithDescription("Kafka topic partition diff gauge."),
		metric.WithUnit("1"))
	if err != nil {
		panic(err)
	}

	messageStatusCount, err = otel.Meter("kafka").Int64Counter("kafka.message.status",
		metric.WithDescription("Kafka message status counter."),
		metric.WithUnit("1"))
	if err != nil {
		panic(err)
	}
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
