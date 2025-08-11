package v2

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/observability"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const packageName = "kafka"

var publishCount metric.Int64Counter

func init() {
	publishCount = patronmetric.Int64Counter(packageName, "kafka.publish.count", "Kafka message count.", "1")
}

func publishCountAdd(ctx context.Context, attrs ...attribute.KeyValue) {
	publishCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func startSpan(ctx context.Context, action, delivery, topic string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("delivery", delivery),
		observability.ClientAttribute("kafka"),
	}

	if topic != "" {
		attrs = append(attrs, attribute.String("topic", topic))
	}

	return patrontrace.StartSpan(ctx, action, trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(attrs...))
}

func injectTracingAndCorrelationHeaders(ctx context.Context, msg *sarama.ProducerMessage) {
	msg.Headers = append(msg.Headers, sarama.RecordHeader{
		Key:   []byte(correlation.HeaderID),
		Value: []byte(correlation.IDFromContext(ctx)),
	})

	otel.GetTextMapPropagator().Inject(ctx, producerMessageCarrier{msg})
}

func topicAttribute(topic string) attribute.KeyValue {
	return attribute.String("topic", topic)
}

type producerMessageCarrier struct {
	msg *sarama.ProducerMessage
}

// Get retrieves a single value for a given key.
func (c producerMessageCarrier) Get(_ string) string {
	return ""
}

// Set sets a header.
func (c producerMessageCarrier) Set(key, val string) {
	c.msg.Headers = append(c.msg.Headers, sarama.RecordHeader{Key: []byte(key), Value: []byte(val)})
}

// Keys returns a slice of all key identifiers in the carrier.
func (c producerMessageCarrier) Keys() []string {
	return nil
}

func statusCountBatchAdd(ctx context.Context, statusAttr attribute.KeyValue, messages []*sarama.ProducerMessage) {
	for _, msg := range messages {
		publishCountAdd(ctx, deliveryTypeSyncAttr, statusAttr, topicAttribute(msg.Topic))
	}
}
