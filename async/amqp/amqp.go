package amqp

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/google/uuid"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/streadway/amqp"
)

var (
	defaultCfg = amqp.Config{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 30*time.Second)
		},
	}
)

type message struct {
	span    opentracing.Span
	ctx     context.Context
	del     *amqp.Delivery
	dec     encoding.DecodeRawFunc
	requeue bool
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

// Exchange represents an AMQP exchange.
type Exchange struct {
	name string
	kind string
}

// NewExchange validates its input and creates a new Exchange.
// name should be a non empty string, kind should be one of: [direct, fanout, topic, headers].
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
		info:     make(map[string]interface{}),
	}

	for _, o := range f.oo {
		err := o(c)
		if err != nil {
			return nil, err
		}
	}

	c.createInfo()
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
	info     map[string]interface{}
}

// Info return the information of the consumer.
func (c *consumer) Info() map[string]interface{} {
	return c.info
}

// Consume starts of consuming a AMQP queue.
func (c *consumer) Consume(ctx context.Context) (<-chan async.Message, <-chan error, error) {
	deliveries, err := c.consume()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed initialize consumer")
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
				sp, chCtx := trace.ConsumerSpan(
					ctx,
					trace.ComponentOpName(trace.AMQPConsumerComponent, c.queue),
					trace.AMQPConsumerComponent,
					mapHeader(d.Headers),
					c.traceTag,
				)
				dec, err := async.DetermineDecoder(d.ContentType)
				if err != nil {
					err := errors.Aggregate(err, errors.Wrap(d.Nack(false, c.requeue), "failed to NACK message"))
					trace.SpanError(sp)
					chErr <- err
					return
				}
				chCtx = log.WithContext(chCtx, log.Sub(map[string]interface{}{"messageID": uuid.New().String()}))
				chMsg <- &message{
					ctx:     chCtx,
					dec:     dec,
					del:     &d,
					span:    sp,
					requeue: c.requeue,
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
		errChan = errors.Wrapf(c.ch.Cancel(c.tag, true), "failed to cancel channel of consumer %s", c.tag)
	}
	if c.conn != nil {
		errConn = errors.Wrap(c.conn.Close(), "failed to close connection")
	}
	return errors.Aggregate(errChan, errConn)
}

func (c *consumer) consume() (<-chan amqp.Delivery, error) {
	conn, err := amqp.DialConfig(c.url, c.cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to dial @ %s", c.url)
	}
	c.conn = conn

	ch, err := c.conn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "failed get channel")
	}
	c.ch = ch

	c.tag = uuid.New().String()
	log.Infof("consuming messages for tag %s", c.tag)

	err = ch.ExchangeDeclare(c.exchange.name, c.exchange.kind, true, false, false, false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to declare exchange")
	}

	q, err := ch.QueueDeclare(c.queue, true, false, false, false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to declare queue")
	}

	for _, binding := range c.bindings {
		if err := ch.QueueBind(q.Name, binding, c.exchange.name, false, nil); err != nil {
			return nil, errors.Wrap(err, "failed to bind queue to exchange queue")
		}
	}

	deliveries, err := ch.Consume(c.queue, c.tag, false, false, false, false, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed initialize consumer")
	}

	return deliveries, nil
}

func (c *consumer) createInfo() {
	c.info["type"] = "amqp-consumer"
	c.info["queue"] = c.queue
	c.info["exchange"] = c.exchange
	c.info["requeue"] = c.requeue
	c.info["buffer"] = c.buffer

	var re = regexp.MustCompile(`(?m)amqp:\/\/\w*:\w*@\w*:\d*\/`)
	match := re.FindAllString(c.url, -1)
	if len(match) > 0 {
		lst := strings.LastIndex(c.url, "@")
		c.info["url"] = "amqp://xxx:xxx" + c.url[lst:]
		return
	}
	c.info["url"] = c.url
}

func mapHeader(hh amqp.Table) map[string]string {
	mp := make(map[string]string)
	for k, v := range hh {
		mp[k] = fmt.Sprint(v)
	}
	return mp
}
