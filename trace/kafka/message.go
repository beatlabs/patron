package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"

	"github.com/opentracing/opentracing-go"
)

// Message abstraction of a Kafka message.
type Message struct {
	Topic string
	Body  interface{}
	Key   *string
}

type kafkaHeadersCarrier []sarama.RecordHeader

// Set implements Set() of opentracing.TextMapWriter.
func (c *kafkaHeadersCarrier) Set(key, val string) {
	*c = append(*c, sarama.RecordHeader{Key: []byte(key), Value: []byte(val)})
}

// NewMessage creates a new message.
func NewMessage(t string, b interface{}) *Message {
	return &Message{Topic: t, Body: b}
}

// NewMessageWithKey creates a new message with an associated key.
func NewMessageWithKey(t string, b interface{}, k string) (*Message, error) {
	if k == "" {
		return nil, errors.New("key string can not be null")
	}
	return &Message{Topic: t, Body: b, Key: &k}, nil
}

// CreateProducerMessage creates a producer message which is used for delivery to the Kafka broker.
func (msg *Message) CreateProducerMessage(ctx context.Context, contentType string, ecf encoding.EncodeFunc, sp opentracing.Span) (*sarama.ProducerMessage, error) {
	c := kafkaHeadersCarrier{}
	err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, &c)
	if err != nil {
		return nil, fmt.Errorf("failed to inject tracing headers: %w", err)
	}
	c.Set(encoding.ContentTypeHeader, contentType)

	var saramaKey sarama.Encoder
	if msg.Key != nil {
		saramaKey = sarama.StringEncoder(*msg.Key)
	}

	b, err := ecf(msg.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message body")
	}

	c.Set(correlation.HeaderID, correlation.IDFromContext(ctx))
	return &sarama.ProducerMessage{
		Topic:   msg.Topic,
		Key:     saramaKey,
		Value:   sarama.ByteEncoder(b),
		Headers: c,
	}, nil
}
