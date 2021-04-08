// Package group provides kafka consumer group component implementation.
package group

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/correlation"
	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/internal/validation"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	consumerComponent = "kafka-consumer"
	subsystem         = "kafka"
	messageReceived   = "received"
	messageProcessed  = "processed"
	messageErrored    = "errored"
	messageSkipped    = "skipped"
)

const (
	defaultRetries         = 3
	defaultRetryWait       = 10 * time.Second
	defaultBatchSize       = 1
	defaultBatchTimeout    = 100 * time.Millisecond
	defaultFailureStrategy = kafka.ExitStrategy
)

var (
	consumerErrors           *prometheus.CounterVec
	topicPartitionOffsetDiff *prometheus.GaugeVec
	messageStatus            *prometheus.CounterVec
)

func init() {
	consumerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: subsystem,
			Name:      "consumer_errors",
			Help:      "Consumer errors, classified by consumer name",
		},
		[]string{"name"},
	)

	topicPartitionOffsetDiff = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "component",
			Subsystem: subsystem,
			Name:      "offset_diff",
			Help:      "Message offset difference with high watermark, classified by topic and partition",
		},
		[]string{"group", "topic", "partition"},
	)

	messageStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: subsystem,
			Name:      "message_status",
			Help:      "Message status counter (received, processed, errored) classified by topic and partition",
		}, []string{"status", "group", "topic"},
	)

	prometheus.MustRegister(
		consumerErrors,
		topicPartitionOffsetDiff,
		messageStatus,
	)
}

// consumerErrorsInc increments the number of errors encountered by a specific consumer
func consumerErrorsInc(name string) {
	consumerErrors.WithLabelValues(name).Inc()
}

// topicPartitionOffsetDiffGaugeSet creates a new Gauge that measures partition offsets.
func topicPartitionOffsetDiffGaugeSet(group, topic string, partition int32, high, offset int64) {
	topicPartitionOffsetDiff.WithLabelValues(group, topic, strconv.FormatInt(int64(partition), 10)).Set(float64(high - offset))
}

// messageStatusCountInc increments the messageStatus counter for a certain status.
func messageStatusCountInc(status, group, topic string) {
	messageStatus.WithLabelValues(status, group, topic).Inc()
}

// New initializes a new  kafka consumer component with support for functional configuration.
// The default failure strategy is the ExitStrategy.
// The default batch size is 1 and the batch timeout is 100ms.
// The default number of retries is 0 and the retry wait is 0.
func New(name, group string, brokers, topics []string, proc kafka.BatchProcessorFunc, oo ...OptionFunc) (*Component, error) {
	var errs []error
	if name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if group == "" {
		errs = append(errs, errors.New("consumer group is required"))
	}

	if validation.IsStringSliceEmpty(brokers) {
		errs = append(errs, errors.New("brokers are empty or have an empty value"))
	}

	if validation.IsStringSliceEmpty(topics) {
		errs = append(errs, errors.New("topics are empty or have an empty value"))
	}

	if proc == nil {
		errs = append(errs, errors.New("work processor is required"))
	}

	if len(errs) > 0 {
		return nil, patronErrors.Aggregate(errs...)
	}

	defaultSaramaCfg, err := kafka.DefaultSaramaConfig(name)
	if err != nil {
		return nil, err
	}

	cmp := &Component{
		name:         name,
		group:        group,
		brokers:      brokers,
		topics:       topics,
		proc:         proc,
		retries:      defaultRetries,
		retryWait:    defaultRetryWait,
		batchSize:    defaultBatchSize,
		batchTimeout: defaultBatchTimeout,
		failStrategy: defaultFailureStrategy,
		saramaConfig: defaultSaramaCfg,
	}

	for _, optionFunc := range oo {
		err = optionFunc(cmp)
		if err != nil {
			return nil, err
		}
	}

	return cmp, nil
}

// Component is a kafka consumer implementation that processes messages in batch
type Component struct {
	name         string
	group        string
	topics       []string
	brokers      []string
	saramaConfig *sarama.Config
	proc         kafka.BatchProcessorFunc
	failStrategy kafka.FailStrategy
	batchSize    uint
	batchTimeout time.Duration
	retries      uint
	retryWait    time.Duration
	commitSync   bool
}

// Run starts the consumer processing loop to process messages from Kafka.
func (c *Component) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	return c.processing(ctx)
}

func (c *Component) processing(ctx context.Context) error {
	var componentError error

	retries := int(c.retries)
	for i := 0; i <= retries; i++ {
		handler := newConsumerHandler(ctx, c.name, c.group, c.proc, c.failStrategy, c.batchSize,
			c.batchTimeout, c.commitSync)

		client, err := sarama.NewConsumerGroup(c.brokers, c.group, c.saramaConfig)
		componentError = err
		if err != nil {
			log.Errorf("error creating consumer group client for kafka component: %v", err)
		}

		if client != nil {
			log.Infof("consuming messages from topics '%#v' using group '%s'", c.topics, c.group)
			for {
				// check if context was cancelled or deadline exceeded, signaling that the consumer should stop
				if ctx.Err() != nil {
					log.Infof("kafka component terminating: context cancelled or deadline exceeded")
					return componentError
				}

				// `Consume` should be called inside an infinite loop, when a
				// server-side re-balance happens, the consumer session will need to be
				// recreated to get the new claims
				err := client.Consume(ctx, c.topics, handler)
				componentError = err
				if err != nil {
					log.Errorf("error from kafka consumer: %v", err)
					break
				}

				if handler.err != nil {
					break
				}
			}

			err = client.Close()
			if err != nil {
				log.Errorf("error closing kafka consumer: %v", err)
			}
		}

		consumerErrorsInc(c.name)

		if c.retries > 0 {
			if handler.processedMessages {
				i = 0
			}

			// if no component error has already been set, it is probably a handler error
			if componentError == nil {
				componentError = handler.err
			}

			log.Errorf("failed run, retry %d/%d with %v wait: %v", i, c.retries, c.retryWait, componentError)
			time.Sleep(c.retryWait)

			if i < retries {
				// set the component error to nil to ready for the next iteration
				componentError = nil
			}
		}

		// If there is no component error which is a result of not being able to initialize the consumer
		// then the handler errored while processing a message. This faulty message is then the reason
		// behind the component failure.
		if i == retries && componentError == nil {
			componentError = fmt.Errorf("message processing failure exhausted %d retries: %w", i, handler.err)
		}
	}

	return componentError
}

// Consumer represents a sarama consumer group consumer
type consumerHandler struct {
	ctx context.Context

	name  string
	group string

	// buffer
	batchSize int
	ticker    *time.Ticker

	// callback
	proc kafka.BatchProcessorFunc

	// failures strategy
	failStrategy kafka.FailStrategy

	// committing after every batch
	commitSync bool

	// lock to protect buffer operation
	mu     sync.RWMutex
	msgBuf []*sarama.ConsumerMessage

	// processing error
	err error

	// whether the handler has processed any messages
	processedMessages bool
}

func newConsumerHandler(ctx context.Context, name, group string, processorFunc kafka.BatchProcessorFunc,
	fs kafka.FailStrategy, batchSize uint, batchTimeout time.Duration, commitSync bool) *consumerHandler {

	return &consumerHandler{
		ctx:          ctx,
		name:         name,
		group:        group,
		batchSize:    int(batchSize),
		ticker:       time.NewTicker(batchTimeout),
		msgBuf:       make([]*sarama.ConsumerMessage, 0, batchSize),
		mu:           sync.RWMutex{},
		proc:         processorFunc,
		failStrategy: fs,
		commitSync:   commitSync,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *consumerHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *consumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if ok {
				log.Debugf("message claimed: value = %s, timestamp = %v, topic = %s", string(msg.Value), msg.Timestamp, msg.Topic)
				topicPartitionOffsetDiffGaugeSet(c.group, msg.Topic, msg.Partition, claim.HighWaterMarkOffset(), msg.Offset)
				messageStatusCountInc(messageReceived, c.group, msg.Topic)
				err := c.insertMessage(session, msg)
				if err != nil {
					return err
				}
			} else {
				log.Debug("messages channel closed")
				return nil
			}
		case <-c.ticker.C:
			c.mu.Lock()
			err := c.flush(session)
			c.mu.Unlock()
			if err != nil {
				return err
			}
		case <-c.ctx.Done():
			if c.ctx.Err() != context.Canceled {
				log.Infof("closing consumer: %v", c.ctx.Err())
			}
			return nil
		}
	}
}

func (c *consumerHandler) flush(session sarama.ConsumerGroupSession) error {
	if len(c.msgBuf) > 0 {
		messages := make([]kafka.Message, 0, len(c.msgBuf))
		for _, msg := range c.msgBuf {
			messageStatusCountInc(messageProcessed, c.group, msg.Topic)
			ctx, sp := c.getContextWithCorrelation(msg)
			messages = append(messages, kafka.NewMessage(ctx, sp, msg))
		}

		btc := kafka.NewBatch(messages)
		err := c.proc(btc)
		if err != nil {
			if c.ctx.Err() == context.Canceled {
				return fmt.Errorf("context was cancelled after processing error: %w", err)
			}
			err := c.executeFailureStrategy(messages, err)
			if err != nil {
				return err
			}
		}

		c.processedMessages = true
		for _, m := range messages {
			trace.SpanSuccess(m.Span())
			session.MarkMessage(m.Message(), "")
		}

		if c.commitSync {
			session.Commit()
		}

		c.msgBuf = c.msgBuf[:0]
	}

	return nil
}

func (c *consumerHandler) executeFailureStrategy(messages []kafka.Message, err error) error {
	switch c.failStrategy {
	case kafka.ExitStrategy:
		for _, m := range messages {
			trace.SpanError(m.Span())
			messageStatusCountInc(messageErrored, c.group, m.Message().Topic)
		}
		log.Errorf("could not process message(s)")
		c.err = err
		return err
	case kafka.SkipStrategy:
		for _, m := range messages {
			trace.SpanError(m.Span())
			messageStatusCountInc(messageErrored, c.group, m.Message().Topic)
			messageStatusCountInc(messageSkipped, c.group, m.Message().Topic)
		}
		log.Errorf("could not process message(s) so skipping with error: %v", err)
	default:
		log.Errorf("unknown failure strategy executed")
		return fmt.Errorf("unknown failure strategy: %v", c.failStrategy)
	}
	return nil
}

func (c *consumerHandler) getContextWithCorrelation(msg *sarama.ConsumerMessage) (context.Context, opentracing.Span) {
	corID := getCorrelationID(msg.Headers)

	sp, ctxCh := trace.ConsumerSpan(c.ctx, trace.ComponentOpName(consumerComponent, msg.Topic),
		consumerComponent, corID, mapHeader(msg.Headers))
	ctxCh = correlation.ContextWithID(ctxCh, corID)
	ctxCh = log.WithContext(ctxCh, log.Sub(map[string]interface{}{correlation.ID: corID}))
	return ctxCh, sp
}

func (c *consumerHandler) insertMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgBuf = append(c.msgBuf, msg)
	if len(c.msgBuf) >= c.batchSize {
		return c.flush(session)
	}
	return nil
}

func getCorrelationID(hh []*sarama.RecordHeader) string {
	for _, h := range hh {
		if string(h.Key) == correlation.HeaderID {
			if len(h.Value) > 0 {
				return string(h.Value)
			}
			break
		}
	}
	log.Debug("correlation header not found, creating new correlation UUID")
	return uuid.New().String()
}

func mapHeader(hh []*sarama.RecordHeader) map[string]string {
	mp := make(map[string]string)
	for _, h := range hh {
		mp[string(h.Key)] = string(h.Value)
	}
	return mp
}
