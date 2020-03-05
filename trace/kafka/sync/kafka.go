package kafka

import (
	"context"
	"fmt"

	"github.com/beatlabs/patron/trace"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Message abstraction of a Kafka message.
type Message struct {
	topic string
	body  []byte
}

// NewMessage creates a new message.
func NewMessage(t string, b []byte) *Message {
	return &Message{topic: t, body: b}
}

// NewJSONMessage creates a new message with a JSON encoded body.
func NewJSONMessage(t string, d interface{}) (*Message, error) {
	b, err := json.Encode(d)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON encode: %w", err)
	}
	return &Message{topic: t, body: b}, nil
}

// Producer interface for Kafka.
type Producer interface {
	Send(ctx context.Context, msg *Message) (int32, int64, error)
	Close() error
}

// SyncProducer defines a sync Kafka producer.
type SyncProducer struct {
	cfg  *sarama.Config
	prod sarama.SyncProducer
	tag  opentracing.Tag
}

// NewSyncProducer creates a new sync producer with default configuration.
// Minimum supported KafkaVersion is 11.0.0, and can be updated with an
// OptionFunc.
func NewSyncProducer(brokers []string, oo ...OptionFunc) (*SyncProducer, error) {

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V0_11_0_0
	cfg.Producer.Return.Successes = true

	sp := SyncProducer{cfg: cfg, tag: opentracing.Tag{Key: "type", Value: "sync"}}

	for _, o := range oo {
		err := o(&sp)
		if err != nil {
			return nil, err
		}
	}

	prod, err := sarama.NewSyncProducer(brokers, sp.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync producer: %w", err)
	}
	sp.prod = prod
	return &sp, nil
}

// Send a message to a topic. It will return the partition and the offset of the
// produced message, or an error if the message failed to produce.
func (sp *SyncProducer) Send(ctx context.Context, msg *Message) (partition int32, offset int64, err error) {
	ts, _ := trace.ChildSpan(
		ctx,
		trace.ComponentOpName(trace.KafkaSyncProducerComponent, msg.topic),
		trace.KafkaSyncProducerComponent,
		ext.SpanKindProducer,
		sp.tag,
		opentracing.Tag{Key: "topic", Value: msg.topic},
	)
	pm, err := createProducerMessage(msg, ts)
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

func createProducerMessage(msg *Message, sp opentracing.Span) (*sarama.ProducerMessage, error) {
	c := kafkaHeadersCarrier{}
	err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, &c)
	if err != nil {
		return nil, fmt.Errorf("failed to inject tracing headers: %w", err)
	}

	return &sarama.ProducerMessage{
		Topic:   msg.topic,
		Key:     nil,
		Value:   sarama.ByteEncoder(msg.body),
		Headers: c,
	}, nil
}

type kafkaHeadersCarrier []sarama.RecordHeader

// Set implements Set() of opentracing.TextMapWriter.
func (c *kafkaHeadersCarrier) Set(key, val string) {
	*c = append(*c, sarama.RecordHeader{Key: []byte(key), Value: []byte(val)})
}
