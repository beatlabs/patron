package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/cloudevents/sdk-go/v2/binding"

	"github.com/Shopify/sarama"
	"github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	messageCreationErrors = "creation-errors"
	messageSendErrors     = "send-errors"
	messageSent           = "sent"
)

var messageStatus *prometheus.CounterVec

// Producer interface for Kafka.
type Producer interface {
	Send(ctx context.Context, msg *Message) error
	SendCloudEvent(ctx context.Context, topic string, msg *cloudevents.Event) error
	Close() error
}

type baseProducer struct {
	cfg         *sarama.Config
	prodClient  sarama.Client
	tag         opentracing.Tag
	enc         encoding.EncodeFunc
	contentType string
	// deliveryType can be 'sync' or 'async'
	deliveryType  string
	messageStatus *prometheus.CounterVec
}

var (
	_ Producer = &AsyncProducer{}
	_ Producer = &SyncProducer{}

	errTopicEmpty = errors.New("topic is empty")
	errBodyIsNil  = errors.New("body is nil")
)

func init() {
	messageStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "kafka_producer",
			Name:      "message_status",
			Help:      "Message status counter (produced, encoded, encoding-errors) classified by topic",
		}, []string{"status", "topic", "type"},
	)

	prometheus.MustRegister(messageStatus)
}

// Message abstraction of a Kafka message.
type Message struct {
	topic string
	body  interface{}
	key   *string
}

// NewMessage creates a new message.
func NewMessage(topic string, body interface{}) (*Message, error) {
	if topic == "" {
		return nil, errTopicEmpty
	}
	if body == nil {
		return nil, errBodyIsNil
	}
	return &Message{topic: topic, body: body}, nil
}

// NewMessageWithKey creates a new message with an associated key.
func NewMessageWithKey(topic string, body interface{}, key string) (*Message, error) {
	if topic == "" {
		return nil, errTopicEmpty
	}
	if body == nil {
		return nil, errBodyIsNil
	}
	if key == "" {
		return nil, errors.New("key is empty")
	}
	return &Message{topic: topic, body: body, key: &key}, nil
}

func (p *baseProducer) statusCountInc(status, topic string) {
	p.messageStatus.WithLabelValues(status, topic, p.deliveryType).Inc()
}

// ActiveBrokers returns a list of active brokers' addresses.
func (p *baseProducer) ActiveBrokers() []string {
	brokers := p.prodClient.Brokers()
	activeBrokerAddresses := make([]string, len(brokers))
	for i, b := range brokers {
		activeBrokerAddresses[i] = b.Addr()
	}
	return activeBrokerAddresses
}

func (p *baseProducer) createProducerMessage(ctx context.Context, msg *Message, sp opentracing.Span) (*sarama.ProducerMessage, error) {
	headersCarrier := kafkaHeadersCarrier{}
	err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, &headersCarrier)
	if err != nil {
		return nil, fmt.Errorf("failed to inject tracing headers: %w", err)
	}
	headersCarrier.Set(correlation.HeaderID, correlation.IDFromContext(ctx))

	headersCarrier.Set(encoding.ContentTypeHeader, p.contentType)

	var key sarama.Encoder
	if msg.key != nil {
		key = sarama.StringEncoder(*msg.key)
	}

	b, err := p.enc(msg.body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message body: %w", err)
	}

	return &sarama.ProducerMessage{
		Topic:   msg.topic,
		Key:     key,
		Value:   sarama.ByteEncoder(b),
		Headers: headersCarrier,
	}, nil
}

func createProducerMessageFromCloudEvent(ctx context.Context, sp opentracing.Span, topic string,
	msg *cloudevents.Event) (*sarama.ProducerMessage, error) {

	headerCarrier := kafkaHeadersCarrier{}
	err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, &headerCarrier)
	if err != nil {
		return nil, fmt.Errorf("failed to inject tracing headers: %w", err)
	}
	headerCarrier.Set(correlation.HeaderID, correlation.IDFromContext(ctx))

	kafkaMsg := &sarama.ProducerMessage{Topic: topic}

	err = kafka_sarama.WriteProducerMessage(ctx, (*binding.EventMessage)(msg), kafkaMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudEventKafka message: %w", err)
	}

	kafkaMsg.Headers = append(kafkaMsg.Headers, headerCarrier...)

	return kafkaMsg, nil
}

type kafkaHeadersCarrier []sarama.RecordHeader

// Set implements Set() of opentracing.TextMapWriter.
func (c *kafkaHeadersCarrier) Set(key, val string) {
	*c = append(*c, sarama.RecordHeader{Key: []byte(key), Value: []byte(val)})
}
