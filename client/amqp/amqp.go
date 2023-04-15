// Package amqp provides a client with included tracing capabilities.
package amqp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/beatlabs/patron/correlation"
	patronerrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/streadway/amqp"
	"golang.org/x/exp/slog"
)

const (
	publisherComponent = "amqp-publisher"
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
		return nil, patronerrors.Aggregate(fmt.Errorf("failed to open channel: %w", err), conn.Close())
	}

	pub.connection = conn
	pub.channel = ch
	return pub, nil
}

// Publish a message to an exchange.
func (tc *Publisher) Publish(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	sp := injectTraceHeaders(ctx, exchange, &msg)

	start := time.Now()
	err := tc.channel.Publish(exchange, key, mandatory, immediate, msg)

	observePublish(ctx, sp, start, exchange, err)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func injectTraceHeaders(ctx context.Context, exchange string, msg *amqp.Publishing) opentracing.Span {
	sp, _ := trace.ChildSpan(ctx, trace.ComponentOpName(publisherComponent, exchange),
		publisherComponent, ext.SpanKindProducer, opentracing.Tag{Key: "exchange", Value: exchange})

	if msg.Headers == nil {
		msg.Headers = amqp.Table{}
	}

	c := amqpHeadersCarrier(msg.Headers)

	if err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, c); err != nil {
		log.FromContext(ctx).Error("failed to inject tracing headers", slog.Any("error", err))
	}
	msg.Headers[correlation.HeaderID] = correlation.IDFromContext(ctx)
	return sp
}

// Close the channel and connection.
func (tc *Publisher) Close() error {
	return patronerrors.Aggregate(tc.channel.Close(), tc.connection.Close())
}

type amqpHeadersCarrier map[string]interface{}

// Set implements Set() of opentracing.TextMapWriter.
func (c amqpHeadersCarrier) Set(key, val string) {
	c[key] = val
}

func observePublish(ctx context.Context, span opentracing.Span, start time.Time, exchange string, err error) {
	trace.SpanComplete(span, err)

	durationHistogram := trace.Histogram{
		Observer: publishDurationMetrics.WithLabelValues(exchange, strconv.FormatBool(err == nil)),
	}
	durationHistogram.Observe(ctx, time.Since(start).Seconds())
}
