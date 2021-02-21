// Package amqp provides a native consumer for the AMQP protocol.
package amqp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/beatlabs/patron/correlation"
	patronerrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/streadway/amqp"
)

const (
	defaultBatchCount        = 1
	defaultBatchTimeout      = 1<<63 - 1 // max time duration possible effectively disabling the timeout.
	defaultHeartbeat         = 10 * time.Second
	defaultConnectionTimeout = 30 * time.Second
	defaultLocale            = "en_US"

	consumerComponent = "amqp-consumer"
)

// ProcessorFunc definition of a async processor.
type ProcessorFunc func(context.Context, Batch)

type batchConfig struct {
	count   uint
	timeout time.Duration
}

// Component implementation of a async component.
type Component struct {
	name     string
	url      string
	queue    string
	requeue  bool
	proc     ProcessorFunc
	batchCfg batchConfig
	cfg      amqp.Config
	traceTag opentracing.Tag
	mu       sync.Mutex
}

// New creates a new component with support for functional configuration.
func New(name, url, queue string, proc ProcessorFunc, oo ...OptionFunc) (*Component, error) {
	if name == "" {
		return nil, errors.New("component name is empty")
	}

	if url == "" {
		return nil, errors.New("url is empty")
	}

	if queue == "" {
		return nil, errors.New("queue is empty")
	}

	if proc == nil {
		return nil, errors.New("process function is nil")
	}

	cmp := &Component{
		name:     name,
		url:      url,
		queue:    queue,
		proc:     proc,
		traceTag: opentracing.Tag{Key: "queue", Value: queue},
		batchCfg: batchConfig{
			count:   defaultBatchCount,
			timeout: defaultBatchTimeout,
		},
		cfg: amqp.Config{
			Heartbeat: defaultHeartbeat,
			Locale:    defaultLocale,
			Dial: func(network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, defaultConnectionTimeout)
			},
		},
		mu: sync.Mutex{},
	}

	var err error

	for _, optionFunc := range oo {
		err = optionFunc(cmp)
		if err != nil {
			return nil, err
		}
	}

	return cmp, nil
}

// Run starts the consumer processing loop messages.
func (c *Component) Run(ctx context.Context) error {
	sub, err := c.subscribe()
	if err != nil {
		return err
	}
	defer func() {
		err := sub.close()
		if err != nil {
			log.Errorf("failed to close amqp channel/connection: %v", err)
		}
	}()

	batchTimeout := time.NewTicker(c.batchCfg.timeout)
	defer batchTimeout.Stop()

	btc := &batch{
		ctx:      ctx,
		messages: make([]Message, 0, c.batchCfg.count),
	}

	for {
		select {
		case <-ctx.Done():
			log.FromContext(ctx).Info("context cancellation received. exiting...")
			return nil
		case delivery := <-sub.deliveries:
			log.Debugf("processing message %d", delivery.DeliveryTag)
			c.processBatch(ctx, c.createMessage(ctx, delivery), btc)
		case <-batchTimeout.C:
			log.Debugf("batch timeout expired, sending batch")
			c.sendBatch(ctx, btc)
		}
	}
}

type subscription struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	deliveries <-chan amqp.Delivery
}

func (s subscription) close() error {
	var ee []error
	if s.channel != nil {
		ee = append(ee, s.channel.Close())
	}
	if s.conn != nil {
		ee = append(ee, s.conn.Close())
	}
	return patronerrors.Aggregate(ee...)
}

func (c *Component) subscribe() (subscription, error) {
	conn, err := amqp.DialConfig(c.url, c.cfg)
	if err != nil {
		return subscription{}, fmt.Errorf("failed to dial @ %s: %w", c.url, err)
	}
	sub := subscription{conn: conn}

	ch, err := conn.Channel()
	if err != nil {
		return subscription{}, patronerrors.Aggregate(conn.Close(), fmt.Errorf("failed get channel: %w", err))
	}
	sub.channel = ch

	tag := uuid.New().String()
	log.Infof("consuming messages for tag %s", tag)

	deliveries, err := ch.Consume(c.queue, tag, false, false, false, false, nil)
	if err != nil {
		return subscription{}, patronerrors.Aggregate(ch.Close(), conn.Close(), fmt.Errorf("failed initialize amqp consumer: %w", err))
	}
	sub.deliveries = deliveries

	return sub, nil
}

func (c *Component) createMessage(ctx context.Context, delivery amqp.Delivery) *message {
	corID := getCorrelationID(delivery.Headers)
	sp, ctxMsg := trace.ConsumerSpan(ctx, trace.ComponentOpName(consumerComponent, c.queue),
		consumerComponent, corID, mapHeader(delivery.Headers), c.traceTag)

	ctxMsg = correlation.ContextWithID(ctxMsg, corID)
	ctxMsg = log.WithContext(ctxMsg, log.Sub(map[string]interface{}{correlation.ID: corID}))

	return &message{
		ctx:     ctxMsg,
		span:    sp,
		msg:     delivery,
		requeue: c.requeue,
	}
}

func (c *Component) processBatch(ctx context.Context, msg *message, btc *batch) {
	c.mu.Lock()
	defer c.mu.Unlock()
	btc.messages = append(btc.messages, msg)

	if len(btc.messages) == int(c.batchCfg.count) {
		c.proc(ctx, btc)
		btc = initBatch(ctx, c.batchCfg.count)
	}
}

func (c *Component) sendBatch(ctx context.Context, btc *batch) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.proc(ctx, btc)
	btc = initBatch(ctx, c.batchCfg.count)
}

func initBatch(ctx context.Context, count uint) *batch {
	return &batch{
		ctx:      ctx,
		messages: make([]Message, 0, count),
	}
}

// func observerMessageAge(queue string, attributes map[string]*string) {
// }
//
// func messageCountInc(queue string, state messageState, count int) {
// 	messageCounter.WithLabelValues(queue, string(state), "false").Add(float64(count))
// }
//
// func messageCountErrorInc(queue string, state messageState, count int) {
// 	messageCounter.WithLabelValues(queue, string(state), "true").Add(float64(count))
// }

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
