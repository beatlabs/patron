package kafka

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/correlation"
	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/internal/validation"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slog"
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
	defaultFailureStrategy = ExitStrategy
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

// consumerErrorsInc increments the number of errors encountered by a specific consumer.
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
func New(name, group string, brokers, topics []string, proc BatchProcessorFunc, saramaCfg *sarama.Config, oo ...OptionFunc) (*Component, error) {
	var errs []error
	if name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if group == "" {
		errs = append(errs, errors.New("consumer group is required"))
	}

	if saramaCfg == nil {
		return nil, errors.New("no Sarama configuration specified")
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
		saramaConfig: saramaCfg,
	}

	for _, optionFunc := range oo {
		err := optionFunc(cmp)
		if err != nil {
			return nil, err
		}
	}

	return cmp, nil
}

// Component is a kafka consumer implementation that processes messages in batch.
type Component struct {
	name                      string
	group                     string
	topics                    []string
	brokers                   []string
	saramaConfig              *sarama.Config
	proc                      BatchProcessorFunc
	failStrategy              FailStrategy
	batchSize                 uint
	batchTimeout              time.Duration
	batchMessageDeduplication bool
	retries                   uint
	retryWait                 time.Duration
	commitSync                bool
	sessionCallback           func(sarama.ConsumerGroupSession) error
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
			c.batchTimeout, c.commitSync, c.batchMessageDeduplication, c.sessionCallback)

		client, err := sarama.NewConsumerGroup(c.brokers, c.group, c.saramaConfig)
		componentError = err
		if err != nil {
			slog.Error("error creating consumer group client for kafka component", slog.Any("error", err))
		}

		if client != nil {
			slog.Debug("consuming messages", slog.Any("topics", c.topics), slog.String("group", c.group))
			for {
				// check if context was cancelled or deadline exceeded, signaling that the consumer should stop
				if ctx.Err() != nil {
					slog.Info("kafka component terminating: context cancelled or deadline exceeded", slog.String("name", c.name))
					return componentError
				}

				// `Consume` should be called inside an infinite loop, when a
				// server-side rebalance happens, the consumer session will need to be
				// recreated to get the new claims
				err := client.Consume(ctx, c.topics, handler)
				componentError = err
				if err != nil {
					slog.Error("failure from kafka consumer", slog.Any("error", err))
					break
				}

				if handler.err != nil {
					break
				}
			}

			err = client.Close()
			if err != nil {
				slog.Error("error closing kafka consumer", slog.Any("error", err))
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

			slog.Error("failed run", slog.Int("current", i), slog.Int("retries", int(c.retries)),
				slog.Duration("wait", c.retryWait), slog.Any("error", componentError))
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

// Consumer represents a Sarama consumer group consumer.
type consumerHandler struct {
	ctx context.Context

	name  string
	group string

	// buffer
	batchSize                 int
	ticker                    *time.Ticker
	batchMessageDeduplication bool

	// callback
	proc BatchProcessorFunc

	// failures strategy
	failStrategy FailStrategy

	// committing after every batch
	commitSync bool

	// lock to protect buffer operation
	mu     sync.RWMutex
	msgBuf []*sarama.ConsumerMessage

	// processing error
	err error

	// whether the handler has processed any messages
	processedMessages bool
	sessionCallback   func(sarama.ConsumerGroupSession) error
}

func newConsumerHandler(ctx context.Context, name, group string, processorFunc BatchProcessorFunc,
	fs FailStrategy, batchSize uint, batchTimeout time.Duration, commitSync, batchMessageDeduplication bool,
	sessionCallback func(sarama.ConsumerGroupSession) error,
) *consumerHandler {
	return &consumerHandler{
		ctx:                       ctx,
		name:                      name,
		group:                     group,
		batchSize:                 int(batchSize),
		batchMessageDeduplication: batchMessageDeduplication,
		ticker:                    time.NewTicker(batchTimeout),
		msgBuf:                    make([]*sarama.ConsumerMessage, 0, batchSize),
		mu:                        sync.RWMutex{},
		proc:                      processorFunc,
		failStrategy:              fs,
		commitSync:                commitSync,
		sessionCallback:           sessionCallback,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
func (c *consumerHandler) Setup(cgs sarama.ConsumerGroupSession) error {
	if c.sessionCallback == nil {
		return nil
	}
	return c.sessionCallback(cgs)
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited.
func (c *consumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if ok {
				slog.Debug("message claimed", slog.String("value", string(msg.Value)),
					slog.Time("timestamp", msg.Timestamp), slog.String("topic", msg.Topic))
				topicPartitionOffsetDiffGaugeSet(c.group, msg.Topic, msg.Partition, claim.HighWaterMarkOffset(), msg.Offset)
				messageStatusCountInc(messageReceived, c.group, msg.Topic)
				err := c.insertMessage(session, msg)
				if err != nil {
					return err
				}
			} else {
				slog.Debug("messages channel closed")
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
			if !errors.Is(c.ctx.Err(), context.Canceled) {
				slog.Info("closing consumer", slog.Any("error", c.ctx.Err()))
			}
			return nil
		}
	}
}

func (c *consumerHandler) flush(session sarama.ConsumerGroupSession) error {
	if len(c.msgBuf) == 0 {
		return nil
	}

	messages := make([]Message, 0, len(c.msgBuf))
	for _, msg := range c.msgBuf {
		messageStatusCountInc(messageProcessed, c.group, msg.Topic)
		ctx, sp := c.getContextWithCorrelation(msg)
		messages = append(messages, NewMessage(ctx, sp, msg))
	}

	if c.batchMessageDeduplication {
		messages = deduplicateMessages(messages)
	}
	btc := NewBatch(messages)
	err := c.proc(btc)
	if err != nil {
		if errors.Is(c.ctx.Err(), context.Canceled) {
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

	return nil
}

func (c *consumerHandler) executeFailureStrategy(messages []Message, err error) error {
	switch c.failStrategy {
	case ExitStrategy:
		for _, m := range messages {
			trace.SpanError(m.Span())
			messageStatusCountInc(messageErrored, c.group, m.Message().Topic)
		}
		slog.Error("could not process message(s)")
		c.err = err
		return err
	case SkipStrategy:
		for _, m := range messages {
			trace.SpanError(m.Span())
			messageStatusCountInc(messageErrored, c.group, m.Message().Topic)
			messageStatusCountInc(messageSkipped, c.group, m.Message().Topic)
		}
		slog.Error("could not process message(s) so skipping with error", slog.Any("error", err))
	default:
		slog.Error("unknown failure strategy executed")
		return fmt.Errorf("unknown failure strategy: %v", c.failStrategy)
	}
	return nil
}

func (c *consumerHandler) getContextWithCorrelation(msg *sarama.ConsumerMessage) (context.Context, opentracing.Span) {
	corID := getCorrelationID(msg.Headers)

	sp, ctxCh := trace.ConsumerSpan(c.ctx, trace.ComponentOpName(consumerComponent, msg.Topic),
		consumerComponent, corID, mapHeader(msg.Headers))
	ctxCh = correlation.ContextWithID(ctxCh, corID)
	ctxCh = log.WithContext(ctxCh, slog.With(slog.String(correlation.ID, corID)))
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
	slog.Debug("correlation header not found, creating new correlation UUID")
	return uuid.New().String()
}

func mapHeader(hh []*sarama.RecordHeader) map[string]string {
	mp := make(map[string]string)
	for _, h := range hh {
		mp[string(h.Key)] = string(h.Value)
	}
	return mp
}

// deduplicateMessages takes a slice of Messages and de-duplicates the messages based on the Key of those messages.
// This function assumes that messages are ordered from old to new, and relies on Kafka ordering guarantees within
// partitions. This is the default behaviour from Kafka unless the Producer altered the partition hashing behaviour in
// a nondeterministic way.
func deduplicateMessages(messages []Message) []Message {
	m := map[string]Message{}
	for _, message := range messages {
		m[string(message.Message().Key)] = message
	}

	deduplicated := make([]Message, 0, len(m))
	for _, message := range m {
		deduplicated = append(deduplicated, message)
	}

	return deduplicated
}