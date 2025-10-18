// Package amqp provides a client with included tracing capabilities.
package amqp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/observability"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const packageName = "amqp"

var publishDurationMetrics metric.Float64Histogram

func init() {
	var err error
	publishDurationMetrics, err = patronmetric.Float64Histogram(packageName, "amqp.publish.duration",
		"AMQP publish duration.", "ms")
	if err != nil {
		panic(err)
	}
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
		if err != nil {
			return nil, fmt.Errorf("failed to open connection: %w", err)
		}
		pub.connection = conn
	} else {
		pub.connection, err = amqp.DialConfig(url, *pub.cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to open connection: %w", err)
		}
		conn = pub.connection
	}

	pub.channel, err = conn.Channel()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to open channel: %w", err), conn.Close())
	}

	return pub, nil
}

// Publish a message to an exchange.
func (tc *Publisher) Publish(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	ctx, sp := injectTraceHeaders(ctx, exchange, &msg)
	defer sp.End()

	start := time.Now()
	err := tc.channel.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)

	observePublish(ctx, start, exchange, err)
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
	msg.Headers[correlation.HeaderID] = correlation.IDFromContext(ctx)

	ctx, sp := patrontrace.StartSpan(ctx, "publish", trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(attribute.String("exchange", exchange), observability.ClientAttribute("amqp")),
	)

	otel.GetTextMapPropagator().Inject(ctx, producerMessageCarrier{msg})

	return ctx, sp
}

// Close the channel and connection.
func (tc *Publisher) Close() error {
	return errors.Join(tc.channel.Close(), tc.connection.Close())
}

func observePublish(ctx context.Context, start time.Time, exchange string, err error) {
	publishDurationMetrics.Record(ctx, time.Since(start).Seconds(),
		metric.WithAttributes(attribute.String("exchange", exchange), observability.StatusAttribute(err)))
}

type producerMessageCarrier struct {
	msg *amqp.Publishing
}

// Get retrieves a single value for a given key.
func (c producerMessageCarrier) Get(_ string) string {
	return ""
}

// Set sets a header.
func (c producerMessageCarrier) Set(key, val string) {
	c.msg.Headers[key] = val
}

// Keys returns a slice of all key identifiers in the carrier.
func (c producerMessageCarrier) Keys() []string {
	return nil
}
