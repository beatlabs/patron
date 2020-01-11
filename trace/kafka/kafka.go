package kafka

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/pkg/errors"
)

// Message abstraction of a Kafka message.
type Message struct {
	topic string
	body  interface{}
	key   *string
}

// NewMessage creates a new message.
func NewMessage(t string, b interface{}) *Message {
	return &Message{topic: t, body: b}
}

// NewMessageWithKey creates a new message with an associated key.
func NewMessageWithKey(t string, b interface{}, k string) (*Message, error) {
	if k == "" {
		return nil, errors.New("key string can not be null")
	}
	return &Message{topic: t, body: b, key: &k}, nil
}

// NewJSONMessage creates a new message with a JSON encoded body.
func NewJSONMessage(t string, d interface{}) (*Message, error) {
	b, err := json.Encode(d)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON encode: %w", err)
	}
	return &Message{topic: t, body: b}, nil
}

// NewJSONMessageWithKey creates a new message with a JSON encoded body and a message key.
func NewJSONMessageWithKey(t string, d interface{}, k string) (*Message, error) {
	if k == "" {
		return nil, errors.New("key string can not be null")
	}
	b, err := json.Encode(d)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON encode: %w", err)
	}
	return &Message{topic: t, body: b, key: &k}, nil
}

// Producer interface for Kafka.
type Producer interface {
	Send(ctx context.Context, msg *Message) error
	Error() <-chan error
	Close() error
}

// AsyncProducer defines a async Kafka producer.
type AsyncProducer struct {
	cfg         *sarama.Config
	prod        sarama.AsyncProducer
	chErr       chan error
	tag         opentracing.Tag
	enc         encoding.EncodeFunc
	contentType string
}

// NewAsyncProducer creates a new async producer with default configuration.
func NewAsyncProducer(brokers []string, oo ...OptionFunc) (*AsyncProducer, error) {

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V0_11_0_0

	ap := AsyncProducer{cfg: cfg, chErr: make(chan error), tag: opentracing.Tag{Key: "type", Value: "async"}, enc: json.Encode, contentType: json.Type}

	for _, o := range oo {
		err := o(&ap)
		if err != nil {
			return nil, err
		}
	}

	prod, err := sarama.NewAsyncProducer(brokers, ap.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create async producer: %w", err)
	}
	ap.prod = prod
	go ap.propagateError()
	return &ap, nil
}

// Send a message to a topic.
func (ap *AsyncProducer) Send(ctx context.Context, msg *Message) error {
	sp, _ := trace.ChildSpan(ctx, trace.ComponentOpName(trace.KafkaAsyncProducerComponent, msg.topic),
		trace.KafkaAsyncProducerComponent, ext.SpanKindProducer, ap.tag,
		opentracing.Tag{Key: "topic", Value: msg.topic})
	pm, err := ap.createProducerMessage(ctx, msg, sp)
	if err != nil {
		trace.SpanError(sp)
		return err
	}
	ap.prod.Input() <- pm
	trace.SpanSuccess(sp)
	return nil
}

// Error returns a chanel to monitor for errors.
func (ap *AsyncProducer) Error() <-chan error {
	return ap.chErr
}

// Close gracefully the producer.
func (ap *AsyncProducer) Close() error {
	err := ap.prod.Close()
	if err != nil {
		return fmt.Errorf("failed to close sync producer: %w", err)
	}
	return nil
}

func (ap *AsyncProducer) propagateError() {
	for pe := range ap.prod.Errors() {
		ap.chErr <- fmt.Errorf("failed to send message: %w", pe)
	}
}

func (ap *AsyncProducer) createProducerMessage(ctx context.Context, msg *Message, sp opentracing.Span) (*sarama.ProducerMessage, error) {
	c := kafkaHeadersCarrier{}
	err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, &c)
	if err != nil {
		return nil, fmt.Errorf("failed to inject tracing headers: %w", err)
	}
	c.Set(encoding.ContentTypeHeader, ap.contentType)

	var saramaKey sarama.Encoder
	if msg.key != nil {
		saramaKey = sarama.StringEncoder(*msg.key)
	}

	b, err := ap.enc(msg.body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message body")
	}

	c.Set(correlation.HeaderID, correlation.IDFromContext(ctx))
	return &sarama.ProducerMessage{
		Topic:   msg.topic,
		Key:     saramaKey,
		Value:   sarama.ByteEncoder(b),
		Headers: c,
	}, nil
}

type kafkaHeadersCarrier []sarama.RecordHeader

// Set implements Set() of opentracing.TextMapWriter.
func (c *kafkaHeadersCarrier) Set(key, val string) {
	*c = append(*c, sarama.RecordHeader{Key: []byte(key), Value: []byte(val)})
}
