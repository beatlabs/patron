package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/internal/validation"
	"github.com/beatlabs/patron/observability"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kotel"
	"github.com/twmb/franz-go/plugin/kslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	packageName       = "kafka"
	deliveryTypeSync  = "sync"
	deliveryTypeAsync = "async"
)

var (
	publishCount metric.Int64Counter

	deliveryTypeSyncAttr  = attribute.String("delivery", deliveryTypeSync)
	deliveryTypeAsyncAttr = attribute.String("delivery", deliveryTypeAsync)
)

func init() {
	publishCount = patronmetric.Int64Counter(packageName, "kafka.publish.count", "Kafka message count.", "1")
}

func publishCountAdd(ctx context.Context, attrs ...attribute.KeyValue) {
	publishCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Producer is a Kafka producer that supports both synchronous and asynchronous message sending.
type Producer struct {
	client *kgo.Client
}

// New creates a new Kafka producer with the specified brokers and franz-go options.
// Kotel hooks for OpenTelemetry tracing and metrics are automatically configured.
func New(brokers []string, opts ...kgo.Opt) (*Producer, error) {
	if validation.IsStringSliceEmpty(brokers) {
		return nil, errors.New("brokers are empty or have an empty value")
	}

	tracer := kotel.NewTracer(kotel.TracerProvider(otel.GetTracerProvider()))
	meter := kotel.NewMeter(kotel.MeterProvider(otel.GetMeterProvider()))
	kotelService := kotel.NewKotel(kotel.WithTracer(tracer), kotel.WithMeter(meter))

	allOpts := make([]kgo.Opt, 0, 3+len(opts))
	allOpts = append(allOpts, kgo.SeedBrokers(brokers...), kgo.WithHooks(kotelService.Hooks()...), kgo.WithLogger(kslog.New(slog.Default())))
	allOpts = append(allOpts, opts...)

	cl, err := kgo.NewClient(allOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &Producer{client: cl}, nil
}

// Send sends one or more messages synchronously. It returns the produced records
// (with Partition and Offset populated) and an aggregated error for any failures.
func (p *Producer) Send(ctx context.Context, records ...*kgo.Record) ([]*kgo.Record, error) {
	if len(records) == 0 {
		return nil, errors.New("records are empty or nil")
	}

	for _, rec := range records {
		injectCorrelationHeader(ctx, rec)
	}

	results := p.client.ProduceSync(ctx, records...)

	produced := make([]*kgo.Record, 0, len(results))
	var errs []error

	for _, r := range results {
		if r.Err != nil {
			errs = append(errs, r.Err)
			publishCountAdd(ctx, deliveryTypeSyncAttr, observability.FailedAttribute, topicAttribute(r.Record.Topic))
		} else {
			produced = append(produced, r.Record)
			publishCountAdd(ctx, deliveryTypeSyncAttr, observability.SucceededAttribute, topicAttribute(r.Record.Topic))
		}
	}

	if len(errs) > 0 {
		return produced, errors.Join(errs...)
	}

	return produced, nil
}

// SendAsync sends a message asynchronously. Any producer errors are sent to the provided error channel.
func (p *Producer) SendAsync(ctx context.Context, rec *kgo.Record, chErr chan<- error) {
	injectCorrelationHeader(ctx, rec)

	p.client.Produce(ctx, rec, func(r *kgo.Record, err error) {
		if err != nil {
			publishCountAdd(context.Background(), deliveryTypeAsyncAttr, observability.FailedAttribute, topicAttribute(r.Topic))
			chErr <- fmt.Errorf("failed to send message: %w", err)
			return
		}
		publishCountAdd(context.Background(), deliveryTypeAsyncAttr, observability.SucceededAttribute, topicAttribute(r.Topic))
	})
}

// Close flushes any buffered records and then shuts down the producer.
func (p *Producer) Close() {
	p.client.Flush(context.Background())
	p.client.Close()
}

func injectCorrelationHeader(ctx context.Context, rec *kgo.Record) {
	rec.Headers = append(rec.Headers, kgo.RecordHeader{
		Key:   correlation.HeaderID,
		Value: []byte(correlation.IDFromContext(ctx)),
	})
}

func topicAttribute(topic string) attribute.KeyValue {
	return attribute.String("topic", topic)
}
