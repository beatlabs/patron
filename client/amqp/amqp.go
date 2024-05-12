// Package amqp provides a client with included tracing capabilities.
package amqp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/beatlabs/patron/correlation"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/prometheus/client_golang/prometheus"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	publisherComponent = "amqp"
)

var publishDurationMetrics *prometheus.HistogramVec

func init() {
	publishDurationMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "client",
			Subsystem: "amqp",
			Name:      "publish_duration_seconds",
			Help:      "AMQP publish completed by the client.",
		},
		[]string{"exchange", "success"},
	)
	prometheus.MustRegister(publishDurationMetrics)
}

// Publisher defines a RabbitMQ publisher with tracing instrumentation.
type Publisher struct {
	cfg        *amqp.Config
	connection *amqp.Connection
	channel    *amqp.Channel
}

// New constructor.
func New(url string, oo ...OptionFunc) (*Publisher, error) {
	if url == "" {
		return nil, errors.New("url is required")
	}

	var err error
	pub := &Publisher{}

	for _, option := range oo {
		err = option(pub)
		if err != nil {
			return nil, err
		}
	}

	var conn *amqp.Connection

	if pub.cfg == nil {
		conn, err = amqp.Dial(url)
	} else {
		conn, err = amqp.DialConfig(url, *pub.cfg)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to open channel: %w", err), conn.Close())
	}

	pub.connection = conn
	pub.channel = ch
	return pub, nil
}

// Publish a message to an exchange.
func (tc *Publisher) Publish(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	// TODO: Metrics

	ctx, sp := injectTraceHeaders(ctx, exchange, &msg)
	defer sp.End()

	start := time.Now()
	err := tc.channel.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)

	observePublish(start, exchange, err)
	if err != nil {
		sp.RecordError(err)
		sp.SetStatus(codes.Error, "error publishing message")
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func injectTraceHeaders(ctx context.Context, exchange string, msg *amqp.Publishing) (context.Context, trace.Span) {
	if msg.Headers == nil {
		msg.Headers = amqp.Table{}
	}

	ctx, sp := patrontrace.Tracer().Start(ctx, "publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("exchange", exchange),
			attribute.String("component", publisherComponent),
		))

	otel.GetTextMapPropagator().Inject(ctx, producerMessageCarrier{msg})

	msg.Headers[correlation.HeaderID] = correlation.IDFromContext(ctx)
	return ctx, sp
}

// Close the channel and connection.
func (tc *Publisher) Close() error {
	return errors.Join(tc.channel.Close(), tc.connection.Close())
}

func observePublish(start time.Time, exchange string, err error) {
	publishDurationMetrics.WithLabelValues(exchange, strconv.FormatBool(err == nil)).Observe(time.Since(start).Seconds())
}

type producerMessageCarrier struct {
	msg *amqp.Publishing
}

// Get retrieves a single value for a given key.
func (c producerMessageCarrier) Get(key string) string {
	for k, v := range c.msg.Headers {
		if k == key {
			return v.(string)
		}
	}
	return ""
}

// Set sets a header.
func (c producerMessageCarrier) Set(key, val string) {
	c.msg.Headers[key] = val
}

// Keys returns a slice of all key identifiers in the carrier.
func (c producerMessageCarrier) Keys() []string {
	out := make([]string, 0, len(c.msg.Headers))
	for k := range c.msg.Headers {
		out = append(out, k)
	}
	return out
}
