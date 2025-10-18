package kafka

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/internal/validation"
	"github.com/beatlabs/patron/observability"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const packageName = "kafka"

var publishCount metric.Int64Counter

func init() {
	var err error
	publishCount, err = patronmetric.Int64Counter(packageName, "kafka.publish.count", "Kafka message count.", "1")
	if err != nil {
		panic(err)
	}
}

func publishCountAdd(ctx context.Context, attrs ...attribute.KeyValue) {
	publishCount.Add(ctx, 1, metric.WithAttributes(attrs...))
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

// New initiates the AsyncProducer/SyncProducer builder chain with the specified Sarama configuration.
func New(brokers []string, saramaConfig *sarama.Config) *Builder {
	var ee []error
	if validation.IsStringSliceEmpty(brokers) {
		ee = append(ee, errors.New("brokers are empty or have an empty value"))
	}
	if saramaConfig == nil {
		ee = append(ee, errors.New("no Sarama configuration specified"))
	}

	return &Builder{
		brokers: brokers,
		errs:    ee,
		cfg:     saramaConfig,
	}
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

// Create a new synchronous producer.
func (b *Builder) Create() (*SyncProducer, error) {
	if len(b.errs) > 0 {
		return nil, errors.Join(b.errs...)
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
func (b *Builder) CreateAsync() (*AsyncProducer, <-chan error, error) {
	if len(b.errs) > 0 {
		return nil, nil, errors.Join(b.errs...)
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

func startSpan(ctx context.Context, action, delivery, topic string) (context.Context, trace.Span) {
	attrs := []attribute.KeyValue{
		attribute.String("delivery", delivery),
		observability.ClientAttribute("kafka"),
	}

	if topic != "" {
		attrs = append(attrs, attribute.String("topic", topic))
	}

	return patrontrace.StartSpan(ctx, action, trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(attrs...))
}

func injectTracingAndCorrelationHeaders(ctx context.Context, msg *sarama.ProducerMessage) {
	msg.Headers = append(msg.Headers, sarama.RecordHeader{
		Key:   []byte(correlation.HeaderID),
		Value: []byte(correlation.IDFromContext(ctx)),
	})

	otel.GetTextMapPropagator().Inject(ctx, producerMessageCarrier{msg})
}

func topicAttribute(topic string) attribute.KeyValue {
	return attribute.String("topic", topic)
}

type producerMessageCarrier struct {
	msg *sarama.ProducerMessage
}

// Get retrieves a single value for a given key.
func (c producerMessageCarrier) Get(_ string) string {
	return ""
}

// Set sets a header.
func (c producerMessageCarrier) Set(key, val string) {
	c.msg.Headers = append(c.msg.Headers, sarama.RecordHeader{Key: []byte(key), Value: []byte(val)})
}

// Keys returns a slice of all key identifiers in the carrier.
func (c producerMessageCarrier) Keys() []string {
	return nil
}
