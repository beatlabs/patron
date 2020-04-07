package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	producerComponent     = "kafka-async-producer"
	messageCreationErrors = "creation-errors"
	messageSendErrors     = "send-errors"
	messageSent           = "sent"
)

var messageStatus *prometheus.CounterVec

// Producer interface for Kafka.
type Producer interface {
	Send(ctx context.Context, msg *Message) error
	Close() error
}

type baseProducer struct {
	cfg         *sarama.Config
	prodClient  sarama.Client
	tag         opentracing.Tag
	enc         encoding.EncodeFunc
	contentType string
}

var (
	_ Producer = &AsyncProducer{}
	_ Producer = &SyncProducer{}
)

// Message abstraction of a Kafka message.
type Message struct {
	topic string
	body  interface{}
	key   *string
}

func messageStatusCountInc(status, topic string) {
	messageStatus.WithLabelValues(status, topic).Inc()
}

func init() {
	messageStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "kafka_async_producer",
			Name:      "message_status",
			Help:      "Message status counter (received, decoded, decoding-errors) classified by topic",
		}, []string{"status", "topic"},
	)
	prometheus.MustRegister(messageStatus)
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

// ActiveBrokers returns a list of active brokers' addresses.
func (ap *baseProducer) ActiveBrokers() []string {
	brokers := ap.prodClient.Brokers()
	activeBrokerAddresses := make([]string, len(brokers))
	for i, b := range brokers {
		activeBrokerAddresses[i] = b.Addr()
	}
	return activeBrokerAddresses
}

func (ap *baseProducer) createProducerMessage(ctx context.Context, msg *Message, sp opentracing.Span) (*sarama.ProducerMessage, error) {
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
		return nil, fmt.Errorf("failed to encode message body: %w", err)
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
