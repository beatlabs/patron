// Package amqp provides a native consumer for the AMQP protocol.
package amqp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/observability/log"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	amqp "github.com/rabbitmq/amqp091-go"
)

type messageState string

const (
	defaultBatchCount        = 1
	defaultBatchTimeout      = 1<<63 - 1 // max time duration possible effectively disabling the timeout.
	defaultHeartbeat         = 10 * time.Second
	defaultConnectionTimeout = 30 * time.Second
	defaultLocale            = "en_US"
	defaultStatsInterval     = 5 * time.Second
	defaultRetryCount        = 10
	defaultRetryDelay        = 5 * time.Second

	consumerComponent = "amqp"

	ackMessageState     messageState = "ACK"
	nackMessageState    messageState = "NACK"
	fetchedMessageState messageState = "FETCHED"
)

var (
	messageAge        *prometheus.GaugeVec
	messageCounterVec *prometheus.CounterVec
	queueSize         *prometheus.GaugeVec
)

func init() {
	messageAge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "component",
			Subsystem: "amqp",
			Name:      "message_age",
			Help:      "Message age based on the AMQP timestamp",
		},
		[]string{"queue"},
	)
	prometheus.MustRegister(messageAge)
	messageCounterVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "amqp",
			Name:      "message_counter",
			Help:      "Message counter by state and error",
		},
		[]string{"queue", "state", "hasError"},
	)
	prometheus.MustRegister(messageCounterVec)
	queueSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "component",
			Subsystem: "amqp",
			Name:      "queue_size",
			Help:      "Queue size reported by AMQP",
		},
		[]string{"queue"},
	)
	prometheus.MustRegister(queueSize)
}

// ProcessorFunc definition of an async processor.
type ProcessorFunc func(context.Context, Batch)

type queueConfig struct {
	url     string
	queue   string
	requeue bool
}

type batchConfig struct {
	count   uint
	timeout time.Duration
}

type retryConfig struct {
	count uint
	delay time.Duration
}

type statsConfig struct {
	interval time.Duration
}

// Component implementation of an async component.
type Component struct {
	queueCfg queueConfig
	proc     ProcessorFunc
	batchCfg batchConfig
	statsCfg statsConfig
	retryCfg retryConfig
	cfg      amqp.Config
	traceTag opentracing.Tag
}

// New creates a new component with support for functional configuration.
func New(url, queue string, proc ProcessorFunc, oo ...OptionFunc) (*Component, error) {
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
		queueCfg: queueConfig{
			url:     url,
			queue:   queue,
			requeue: true,
		},
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
		statsCfg: statsConfig{
			interval: defaultStatsInterval,
		},
		retryCfg: retryConfig{
			count: defaultRetryCount,
			delay: defaultRetryDelay,
		},
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
	count := c.retryCfg.count

	var err error

	for count > 0 {
		sub, err := c.subscribe()
		if err != nil {
			slog.Warn("failed to subscribe to queue, reconnecting", log.ErrorAttr(err),
				slog.Duration("retry", c.retryCfg.delay))
			time.Sleep(c.retryCfg.delay)
			count--
			continue
		}
		count = c.retryCfg.count

		err = c.processLoop(ctx, sub)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			closeSubscription(sub)
			return nil
		}
		slog.Warn("process loop failure, reconnecting", log.ErrorAttr(err), slog.Duration("retry", c.retryCfg.delay))
		time.Sleep(c.retryCfg.delay)
		count--
		closeSubscription(sub)
	}
	return err
}

func closeSubscription(sub subscription) {
	err := sub.close()
	if err != nil {
		slog.Error("failed to close amqp channel/connection", log.ErrorAttr(err))
	}
	slog.Debug("amqp subscription closed")
}

func (c *Component) processLoop(ctx context.Context, sub subscription) error {
	batchTimeout := time.NewTicker(c.batchCfg.timeout)
	defer batchTimeout.Stop()
	tickerStats := time.NewTicker(c.statsCfg.interval)
	defer tickerStats.Stop()

	btc := &batch{messages: make([]Message, 0, c.batchCfg.count)}

	for {
		select {
		case <-ctx.Done():
			slog.Info("context cancellation received. exiting...")
			return ctx.Err()
		case delivery, ok := <-sub.deliveries:
			if !ok {
				return errors.New("subscription channel closed")
			}
			slog.Debug("processing message", slog.Int64("tag", int64(delivery.DeliveryTag)))
			observeReceivedMessageStats(c.queueCfg.queue, delivery.Timestamp)
			c.processBatch(ctx, c.createMessage(ctx, delivery), btc)
		case <-batchTimeout.C:
			slog.Debug("batch timeout expired, sending batch")
			c.sendBatch(ctx, btc)
		case <-tickerStats.C:
			err := c.stats(sub)
			if err != nil {
				slog.Error("failed to report sqsAPI stats: %v", log.ErrorAttr(err))
			}
		}
	}
}

func observeReceivedMessageStats(queue string, timestamp time.Time) {
	messageAge.WithLabelValues(queue).Set(time.Now().UTC().Sub(timestamp).Seconds())
	messageCountInc(queue, fetchedMessageState, nil)
}

type subscription struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	deliveries <-chan amqp.Delivery
	closed     bool
}

func (s *subscription) close() error {
	if s.closed {
		return nil
	}
	var ee []error
	if s.channel != nil {
		ee = append(ee, s.channel.Close())
	}
	if s.conn != nil {
		ee = append(ee, s.conn.Close())
	}
	s.closed = true
	return errors.Join(ee...)
}

func (c *Component) subscribe() (subscription, error) {
	conn, err := amqp.DialConfig(c.queueCfg.url, c.cfg)
	if err != nil {
		return subscription{}, fmt.Errorf("failed to dial @ %s: %w", c.queueCfg.url, err)
	}
	sub := subscription{conn: conn}

	ch, err := conn.Channel()
	if err != nil {
		return subscription{}, errors.Join(conn.Close(), fmt.Errorf("failed get channel: %w", err))
	}
	sub.channel = ch

	tag := uuid.New().String()
	slog.Debug("consuming messages", slog.String("tag", tag))

	deliveries, err := ch.Consume(c.queueCfg.queue, tag, false, false, false, false, nil)
	if err != nil {
		return subscription{}, errors.Join(ch.Close(), conn.Close(), fmt.Errorf("failed initialize amqp consumer: %w", err))
	}
	sub.deliveries = deliveries

	return sub, nil
}

func (c *Component) createMessage(ctx context.Context, delivery amqp.Delivery) *message {
	corID := getCorrelationID(delivery.Headers)
	ctx, sp := patrontrace.Tracer().Start(ctx, patrontrace.ComponentOpName(consumerComponent, c.queueCfg.queue))

	ctx = correlation.ContextWithID(ctx, corID)
	ctx = log.WithContext(ctx, slog.With(slog.String(correlation.ID, corID)))

	return &message{
		ctx:     ctx,
		span:    sp,
		msg:     delivery,
		requeue: c.queueCfg.requeue,
		queue:   c.queueCfg.queue,
	}
}

func (c *Component) processBatch(ctx context.Context, msg *message, btc *batch) {
	btc.append(msg)

	if len(btc.messages) >= int(c.batchCfg.count) {
		c.processAndResetBatch(ctx, btc)
	}
}

func (c *Component) sendBatch(ctx context.Context, btc *batch) {
	c.processAndResetBatch(ctx, btc)
}

func (c *Component) processAndResetBatch(ctx context.Context, btc *batch) {
	c.proc(ctx, btc)
	btc.reset()
}

func (c *Component) stats(sub subscription) error {
	q, err := sub.channel.QueueInspect(c.queueCfg.queue)
	if err != nil {
		return err
	}

	queueSize.WithLabelValues(c.queueCfg.queue).Set(float64(q.Messages))
	return nil
}

func messageCountInc(queue string, state messageState, err error) {
	hasError := "false"
	if err != nil {
		hasError = "true"
	}
	messageCounterVec.WithLabelValues(queue, string(state), hasError).Inc()
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
