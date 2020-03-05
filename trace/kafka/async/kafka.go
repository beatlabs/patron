package kafka

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/encoding"
	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/trace"
	"github.com/beatlabs/patron/trace/kafka"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	producerComponent     = "kafka-async-producer"
	messageCreationErrors = "creation-errors"
	messageSendErrors     = "send-errors"
	messageSent           = "sent"
)

var messageStatus *prometheus.CounterVec

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

// Producer interface for Kafka.
type Producer interface {
	Send(ctx context.Context, msg *kafka.Message) error
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

// AsyncBuilder is a builder which constructs AsyncProducers.
type AsyncBuilder struct {
	*kafka.Builder
}

// Create constructs the AsyncProducer component by applying the gathered properties.
func (ab *AsyncBuilder) Create() (*AsyncProducer, error) {

	if len(ab.Errors) > 0 {
		return nil, patronErrors.Aggregate(ab.Errors...)
	}

	prod, err := sarama.NewAsyncProducer(ab.Brokers, ab.Cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create async producer: %w", err)
	}

	ap := AsyncProducer{
		cfg:         ab.Cfg,
		prod:        prod,
		chErr:       make(chan error),
		enc:         ab.Enc,
		contentType: ab.ContentType,
		tag:         opentracing.Tag{Key: "type", Value: "async"},
	}

	go ap.propagateError()
	return &ap, nil
}

// Send a message to a topic.
func (ap *AsyncProducer) Send(ctx context.Context, msg *kafka.Message) error {
	sp, _ := trace.ChildSpan(ctx, trace.ComponentOpName(producerComponent, msg.Topic),
		producerComponent, ext.SpanKindProducer, ap.tag,
		opentracing.Tag{Key: "topic", Value: msg.Topic})
	pm, err := msg.CreateProducerMessage(ctx, ap.contentType, ap.enc, sp)
	if err != nil {
		messageStatusCountInc(messageCreationErrors, msg.Topic)
		trace.SpanError(sp)
		return err
	}
	messageStatusCountInc(messageSent, msg.Topic)
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
		messageStatusCountInc(messageSendErrors, pe.Msg.Topic)
		ap.chErr <- fmt.Errorf("failed to send message: %w", pe)
	}
}
