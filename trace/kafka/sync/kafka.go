package kafka

import (
	"context"
	"fmt"

	"github.com/beatlabs/patron/encoding"
	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/trace"
	"github.com/beatlabs/patron/trace/kafka"

	"github.com/Shopify/sarama"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const producerComponent = "kafka-sync-producer"

// Producer interface for Kafka.
type Producer interface {
	Send(ctx context.Context, msg *kafka.Message) (int32, int64, error)
	Close() error
}

// SyncProducer defines a sync Kafka producer.
type SyncProducer struct {
	cfg         *sarama.Config
	prod        sarama.SyncProducer
	tag         opentracing.Tag
	enc         encoding.EncodeFunc
	contentType string
}

// SyncBuilder is a builder which constructs SyncProducers.
type SyncBuilder struct {
	*kafka.Builder
}

// Create constructs the SyncProducer component by applying the gathered properties.
func (b *SyncBuilder) Create() (*SyncProducer, error) {
	if len(b.Errors) > 0 {
		return nil, patronErrors.Aggregate(b.Errors...)
	}

	prod, err := sarama.NewSyncProducer(b.Brokers, b.Cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create async producer: %w", err)
	}

	ap := SyncProducer{
		cfg:         b.Cfg,
		prod:        prod,
		enc:         b.Enc,
		contentType: b.ContentType,
		tag:         opentracing.Tag{Key: "type", Value: "sync"},
	}

	return &ap, nil
}

// Send a message to a topic. It will return the partition and the offset of the
// produced message, or an error if the message failed to produce.
func (sp *SyncProducer) Send(ctx context.Context, msg *kafka.Message) (partition int32, offset int64, err error) {
	ts, _ := trace.ChildSpan(
		ctx,
		trace.ComponentOpName(producerComponent, msg.Topic),
		producerComponent,
		ext.SpanKindProducer,
		sp.tag,
		opentracing.Tag{Key: "topic", Value: msg.Topic},
	)
	pm, err := msg.CreateProducerMessage(ctx, sp.contentType, sp.enc, ts)
	if err != nil {
		trace.SpanError(ts)
		return -1, -1, err
	}

	partition, offset, err = sp.prod.SendMessage(pm)
	switch err {
	case nil:
		trace.SpanSuccess(ts)
	default:
		trace.SpanError(ts)
	}
	return partition, offset, err
}

// Close gracefully the producer.
func (sp *SyncProducer) Close() error {
	err := sp.prod.Close()
	if err != nil {
		return fmt.Errorf("failed to close sync producer: %w", err)
	}
	return nil
}
