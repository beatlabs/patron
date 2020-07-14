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
	asyncProducerComponent = "kafka-async-producer"
	syncProducerComponent  = "kafka-sync-producer"
	messageCreationErrors  = "creation-errors"
	messageSendErrors      = "send-errors"
	messageSent            = "sent"
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
	topic    string
	body     interface{}
	key      *string
	cloudEvt *cloudevents.Event
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

func MessageFromCloudEvent(topic string, evt *cloudevents.Event) (*Message, error) {
	if topic == "" {
		return nil, errTopicEmpty
	}
	if evt == nil {
		return nil, errors.New("cloud event is nil")
	}

	return &Message{topic: topic, cloudEvt: evt}, nil
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

	if msg.cloudEvt == nil {
		return p.createProducerMessageBase(headersCarrier, msg)
	}
	return createProducerMessageFromCloudEvent(ctx, headersCarrier, msg)
}

func (p *baseProducer) createProducerMessageBase(headerCarrier kafkaHeadersCarrier,
	msg *Message) (*sarama.ProducerMessage, error) {

	headerCarrier.Set(encoding.ContentTypeHeader, p.contentType)

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
		Headers: headerCarrier,
	}, nil
}

func createProducerMessageFromCloudEvent(ctx context.Context, headerCarrier kafkaHeadersCarrier,
	msg *Message) (*sarama.ProducerMessage, error) {

	kafkaMsg := &sarama.ProducerMessage{Topic: msg.topic}

	err := kafka_sarama.WriteProducerMessage(ctx, (*binding.EventMessage)(msg.cloudEvt), kafkaMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to create CloudEventKafka message: %w", err)
	}

	kafkaMsg.Headers = append(kafkaMsg.Headers, headerCarrier...)

	return nil, nil
}

type kafkaHeadersCarrier []sarama.RecordHeader

// Set implements Set() of opentracing.TextMapWriter.
func (c *kafkaHeadersCarrier) Set(key, val string) {
	*c = append(*c, sarama.RecordHeader{Key: []byte(key), Value: []byte(val)})
}
