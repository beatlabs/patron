package v2

import (
	"errors"
	"fmt"
	"os"

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

	deliveryStatusSent      deliveryStatus = "sent"
	deliveryStatusSendError deliveryStatus = "send-errors"

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

func statusCountAdd(deliveryType string, status deliveryStatus, topic string, cnt int) {
	messageStatus.WithLabelValues(string(status), topic, deliveryType).Add(float64(cnt))
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

// New initiates the AsyncProducer/SyncProducer builder chain without any Sarama configuration.
// WithConfig must be called before the call to Build.
func New(brokers []string) *Builder {
	var ee []error
	if validation.IsStringSliceEmpty(brokers) {
		ee = append(ee, errors.New("brokers are empty or have an empty value"))
	}

	return &Builder{
		brokers: brokers,
		errs:    ee,
	}
}

// DefaultConsumerSaramaConfig function creates a Sarama configuration with a client ID derived from host name and consumer name.
func DefaultConsumerSaramaConfig(name string, readCommitted bool) (*sarama.Config, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, errors.New("failed to get hostname")
	}

	config := sarama.NewConfig()
	config.ClientID = fmt.Sprintf("%s-%s", host, name)
	config.Consumer.Return.Errors = true
	config.Version = sarama.V0_11_0_0
	if readCommitted {
		// from Kafka documentation:
		// Transactions were introduced in Kafka 0.11.0 wherein applications can write to multiple topics and partitions atomically. In order for this to work, consumers reading from these partitions should be configured to only read committed data. This can be achieved by by setting the isolation.level=read_committed in the consumer's configuration.
		// In read_committed mode, the consumer will read only those transactional messages which have been successfully committed. It will continue to read non-transactional messages as before. There is no client-side buffering in read_committed mode. Instead, the end offset of a partition for a read_committed consumer would be the offset of the first message in the partition belonging to an open transaction. This offset is known as the 'Last Stable Offset'(LSO).
		config.Consumer.IsolationLevel = sarama.ReadCommitted
	}

	return config, nil
}

// DefaultProducerSaramaConfig creates a default Sarama configuration with idempotency enabled.
// See also:
// * https://pkg.go.dev/github.com/Shopify/sarama#RequiredAcks
// * https://pkg.go.dev/github.com/Shopify/sarama#Config
func DefaultProducerSaramaConfig(name string, idempotent bool) (*sarama.Config, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, errors.New("failed to get hostname")
	}

	cfg := sarama.NewConfig()
	cfg.ClientID = fmt.Sprintf("%s-%s", host, name)

	if idempotent {
		cfg.Net.MaxOpenRequests = 1
		cfg.Producer.Idempotent = true
	}
	cfg.Producer.RequiredAcks = sarama.WaitForAll

	return cfg, nil
}

// WithConfig allows to pass into the builder the Sarama configuration.
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

	if b.cfg == nil {
		return nil, errors.New("no Sarama configuration specified")
	}

	// required for any SyncProducer; 'Errors' is already true by default for both async/sync producers
	b.cfg.Producer.Return.Successes = true

	p := SyncProducer{}

	var err error
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
	if b.cfg == nil {
		return nil, nil, errors.New("no Sarama configuration specified")
	}

	ap := &AsyncProducer{
		baseProducer: baseProducer{},
		asyncProd:    nil,
	}

	var err error
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
