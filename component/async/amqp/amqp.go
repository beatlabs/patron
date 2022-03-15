// Package amqp provides consumer implementation with included tracing capabilities.
//
// Deprecated: The AMQP consumer package along with the async component is superseded by the standalone `github.com/beatlabs/component/amqp` package.
// Please refer to the documents and the examples for the usage.
//
// This package is frozen and no new functionality will be added.
package amqp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/streadway/amqp"
)

const (
	consumerComponent = "amqp-consumer"
)

var defaultCfg = amqp.Config{
	Dial: func(network, addr string) (net.Conn, error) {
		return net.DialTimeout(network, addr, 30*time.Second)
	},
}

type message struct {
	span    opentracing.Span
	ctx     context.Context
	del     *amqp.Delivery
	dec     encoding.DecodeRawFunc
	requeue bool
	source  string
}

func (m *message) Context() context.Context {
	return m.ctx
}

func (m *message) Decode(v interface{}) error {
	return m.dec(m.del.Body, v)
}

func (m *message) Ack() error {
	err := m.del.Ack(false)
	trace.SpanSuccess(m.span)
	return err
}

func (m *message) Nack() error {
	err := m.del.Nack(false, m.requeue)
	trace.SpanError(m.span)
	return err
}

// Source returns the queue's name where the message arrived.
func (m *message) Source() string {
	return m.source
}

// Payload returns the message payload.
func (m *message) Payload() []byte {
	return m.del.Body
}

// Raw returns tha AMQP message.
func (m *message) Raw() interface{} {
	return m.del
}

// Exchange represents an AMQP exchange.
type Exchange struct {
	name string
	kind string
}

// NewExchange validates its input and creates a new Exchange.
// name should be a non-empty string, kind should be one of: [direct, fanout, topic, headers].
//
// Deprecated: The AMQP consumer package along with the async component is superseded by the standalone `github.com/beatlabs/component/amqp` package.
func NewExchange(name, kind string) (*Exchange, error) {
	if name == "" {
		return nil, errors.New("AMQP Exchange name is required")
	}

	if kind == "" {
		return nil, errors.New("AMQP Exchange type is required")
	}

	if kind != amqp.ExchangeDirect &&
		kind != amqp.ExchangeFanout &&
		kind != amqp.ExchangeTopic &&
		kind != amqp.ExchangeHeaders {
		return nil, fmt.Errorf("AMQP Exchange type is invalid, one of [%s, %s, %s, %s] is required",
			amqp.ExchangeDirect,
			amqp.ExchangeFanout,
			amqp.ExchangeTopic,
			amqp.ExchangeHeaders)
	}

	return &Exchange{name: name, kind: kind}, nil
}

// Factory of an AMQP consumer.
type Factory struct {
	url      string
	queue    string
	exchange Exchange
	bindings []string
	oo       []OptionFunc
}

// New constructor.
//
// Deprecated: The AMQP consumer package along with the async component is superseded by the standalone `github.com/beatlabs/component/amqp` package.
func New(url, queue string, exchange Exchange, oo ...OptionFunc) (*Factory, error) {
	if url == "" {
		return nil, errors.New("AMQP url is required")
	}

	if queue == "" {
		return nil, errors.New("AMQP queue name is required")
	}

	return &Factory{url: url, queue: queue, exchange: exchange, bindings: []string{""}, oo: oo}, nil
}

// Create a new consumer.
func (f *Factory) Create() (async.Consumer, error) {
	c := &consumer{
		url:      f.url,
		queue:    f.queue,
		exchange: f.exchange,
		requeue:  true,
		cfg:      defaultCfg,
		buffer:   1000,
		traceTag: opentracing.Tag{Key: "queue", Value: f.queue},
	}

	for _, o := range f.oo {
		err := o(c)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

type consumer struct {
	url      string
	queue    string
	exchange Exchange
	bindings []string
	requeue  bool
	tag      string
	buffer   int
	traceTag opentracing.Tag
	cfg      amqp.Config
	ch       *amqp.Channel
	conn     *amqp.Connection
}

func (c *consumer) OutOfOrder() bool {
	return true
}

// Consume starts of consuming a AMQP queue.
func (c *consumer) Consume(ctx context.Context) (<-chan async.Message, <-chan error, error) {
	deliveries, err := c.consume()
	if err != nil {
		return nil, nil, fmt.Errorf("failed initialize consumer: %w", err)
	}

	chMsg := make(chan async.Message, c.buffer)
	chErr := make(chan error, c.buffer)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info("canceling consuming messages requested")
				return
			case d := <-deliveries:
				log.Debugf("processing message %d", d.DeliveryTag)
				corID := getCorrelationID(d.Headers)

				sp, ctxCh := trace.ConsumerSpan(ctx, trace.ComponentOpName(consumerComponent, c.queue),
					consumerComponent, corID, mapHeader(d.Headers), c.traceTag)

				dec, err := async.DetermineDecoder(d.ContentType)
				if err != nil {
					errNack := d.Nack(false, c.requeue)
					if errNack != nil {
						err = patronErrors.Aggregate(err, fmt.Errorf("failed to NACK message: %w", errNack))
					}
					trace.SpanError(sp)
					chErr <- err
					return
				}

				ctxCh = correlation.ContextWithID(ctxCh, corID)
				ctxCh = log.WithContext(ctxCh, log.Sub(map[string]interface{}{correlation.ID: corID}))

				chMsg <- &message{
					ctx:     ctxCh,
					dec:     dec,
					del:     &d,
					span:    sp,
					requeue: c.requeue,
					source:  c.queue,
				}
			}
		}
	}()

	return chMsg, chErr, nil
}

// Close handles closing channel and connection of AMQP.
func (c *consumer) Close() error {
	var errChan error
	var errConn error

	if c.ch != nil {
		err := c.ch.Cancel(c.tag, true)
		if err != nil {
			errChan = fmt.Errorf("failed to cancel channel of consumer %s: %w", c.tag, err)
		}
	}
	if c.conn != nil {
		err := c.conn.Close()
		if err != nil {
			errConn = fmt.Errorf("failed to close connection: %w", err)
		}
	}
	return patronErrors.Aggregate(errChan, errConn)
}

func (c *consumer) consume() (<-chan amqp.Delivery, error) {
	conn, err := amqp.DialConfig(c.url, c.cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to dial @ %s: %w", c.url, err)
	}
	c.conn = conn

	ch, err := c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed get channel: %w", err)
	}
	c.ch = ch

	c.tag = uuid.New().String()
	log.Debugf("consuming messages for tag %s", c.tag)

	err = ch.ExchangeDeclare(c.exchange.name, c.exchange.kind, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	q, err := ch.QueueDeclare(c.queue, true, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	for _, binding := range c.bindings {
		if err := ch.QueueBind(q.Name, binding, c.exchange.name, false, nil); err != nil {
			return nil, fmt.Errorf("failed to bind queue to exchange queue: %w", err)
		}
	}

	deliveries, err := ch.Consume(c.queue, c.tag, false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed initialize consumer: %w", err)
	}

	return deliveries, nil
}

func mapHeader(hh amqp.Table) map[string]string {
	mp := make(map[string]string)
	for k, v := range hh {
		mp[k] = fmt.Sprint(v)
	}
	return mp
}

func getCorrelationID(hh amqp.Table) string {
	for key, value := range hh {
		if key == correlation.HeaderID {
			val, ok := value.(string)
			if ok && val != "" {
				return val
			}
			break
		}
	}
	return uuid.New().String()
}
