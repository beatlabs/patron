package v2

import (
	"errors"
	"fmt"

	"github.com/Shopify/sarama"
	patronerrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/internal/validation"
	"github.com/prometheus/client_golang/prometheus"
)

type (
	deliveryStatus string
)

const (
	deliveryTypeSync  = "sync"
	deliveryTypeAsync = "async"

	deliveryStatusSent          deliveryStatus = "sent"
	deliveryStatusCreationError deliveryStatus = "creation-errors"
	deliveryStatusSendError     deliveryStatus = "send-errors"

	componentTypeAsync = "kafka-async-producer"
	componentTypeSync  = "kafka-sync-producer"
)

var messageStatus *prometheus.CounterVec

func init() {
	messageStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "client",
			Subsystem: "kafka_producer",
			Name:      "message_status",
			Help:      "Message status counter (produced, encoded, encoding-errors) classified by topic",
		}, []string{"status", "topic", "type"},
	)

	prometheus.MustRegister(messageStatus)
}

func statusCountInc(deliveryType string, status deliveryStatus, topic string) {
	messageStatus.WithLabelValues(string(status), topic, deliveryType).Inc()
}

type baseProducer struct {
	prodClient sarama.Client
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

// Builder definition for creating sync and async producers.
type Builder struct {
	brokers []string
	cfg     *sarama.Config
	errs    []error
}

// New initiates the AsyncProducer/SyncProducer builder chain with the default sarama configuration.
func New(brokers []string) *Builder {
	var ee []error
	if validation.IsStringSliceEmpty(brokers) {
		ee = append(ee, errors.New("brokers are empty or have an empty value"))
	}

	return &Builder{
		brokers: brokers,
		cfg:     sarama.NewConfig(),
		errs:    ee,
	}
}

// WithConfig allows to pass into the builder a custom sarama configuration.
func (b *Builder) WithConfig(cfg *sarama.Config) *Builder {
	if cfg == nil {
		b.errs = append(b.errs, errors.New("config is nil"))
		return b
	}
	b.cfg = cfg
	return b
}

// Create a new synchronous producer.
func (b *Builder) Create() (*SyncProducer, error) {
	if len(b.errs) > 0 {
		return nil, patronerrors.Aggregate(b.errs...)
	}

	var err error

	// required for any SyncProducer; 'Errors' is already true by default for both async/sync producers
	b.cfg.Producer.Return.Successes = true

	p := SyncProducer{}

	p.prodClient, err = sarama.NewClient(b.brokers, b.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer client: %w", err)
	}

	p.syncProd, err = sarama.NewSyncProducerFromClient(p.prodClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync producer: %w", err)
	}

	return &p, nil
}

// CreateAsync a new asynchronous producer.
func (b Builder) CreateAsync() (*AsyncProducer, <-chan error, error) {
	if len(b.errs) > 0 {
		return nil, nil, patronerrors.Aggregate(b.errs...)
	}
	var err error

	ap := &AsyncProducer{
		baseProducer: baseProducer{},
		asyncProd:    nil,
	}

	ap.prodClient, err = sarama.NewClient(b.brokers, b.cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create producer client: %w", err)
	}

	ap.asyncProd, err = sarama.NewAsyncProducerFromClient(ap.prodClient)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create async producer: %w", err)
	}
	chErr := make(chan error)
	go ap.propagateError(chErr)

	return ap, chErr, nil
}

type kafkaHeadersCarrier []sarama.RecordHeader

// Set implements Set() of opentracing.TextMapWriter.
func (c *kafkaHeadersCarrier) Set(key, val string) {
	*c = append(*c, sarama.RecordHeader{Key: []byte(key), Value: []byte(val)})
}
