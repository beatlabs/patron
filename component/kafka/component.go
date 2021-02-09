// Package kafka provides kafka component implementation.
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
	"github.com/prometheus/client_golang/prometheus"
)

const (
	propSetMSG        = "property '%s' set for '%s'"
	consumerComponent = "kafka-consumer"
	subsystem         = "kafka"
	messageReceived   = "received"
	messageProcessed  = "processed"
	messageErrored    = "errored"
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

// Builder gathers all required properties in order to construct a batch consumer component
type Builder struct {
	name         string
	group        string
	brokers      []string
	topics       []string
	saramaConfig *sarama.Config
	proc         BatchProcessorFunc
	failStrategy FailStrategy
	batchSize    uint
	batchTimeout time.Duration
	retries      uint
	retryWait    time.Duration
	commitSync   bool
	errors       []error // builder errors
}

// New initializes a new builder for a kafka consumer component with the given name
// by default the failStrategy will be ExitStrategy.
// Also by default the batch size is 1 and the batch timeout is 100ms.
func New(name, group string, brokers, topics []string, proc BatchProcessorFunc) *Builder {
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

	return &Builder{
		name:         name,
		group:        group,
		brokers:      brokers,
		topics:       topics,
		proc:         proc,
		errors:       errs,
		batchSize:    1,
		batchTimeout: 100 * time.Millisecond,
	}
}

// WithFailureStrategy defines the failure strategy to be used
// default value is ExitStrategy
// it will append an error to the builder if the strategy is not one of the pre-defined ones.
func (cb *Builder) WithFailureStrategy(fs FailStrategy) *Builder {
	if fs > SkipStrategy || fs < ExitStrategy {
		cb.errors = append(cb.errors, errors.New("invalid failure strategy provided"))
	} else {
		log.Infof(propSetMSG, "failure strategy", cb.name)
		cb.failStrategy = fs
	}
	return cb
}

// WithRetries specifies the number of retries to be executed per failed processing operation
// default value is '0'.
func (cb *Builder) WithRetries(retries uint) *Builder {
	log.Infof(propSetMSG, "retries", cb.name)
	cb.retries = retries
	return cb
}

// WithRetryWait specifies the duration for the component to wait between retrying processing of messages
// default value is '0'
// it will append an error to the builder if the value is smaller than '0'.
func (cb *Builder) WithRetryWait(retryWait time.Duration) *Builder {
	if retryWait < 0 {
		cb.errors = append(cb.errors, errors.New("invalid retry wait provided"))
	} else {
		log.Infof(propSetMSG, "retryWait", cb.name)
		cb.retryWait = retryWait
	}
	return cb
}

// WithBatching specifies the batch size and timeout of the Kafka consumer component.
// If the size is reached then the batch of messages is processed. Otherwise if the timeout elapses
// without new messages coming in, the messages in the buffer would get processed as a batch.
func (cb *Builder) WithBatching(size uint, timeout time.Duration) *Builder {
	if size == 0 {
		cb.errors = append(cb.errors, errors.New("zero batch size provided"))
	} else {
		log.Infof(propSetMSG, "batchSize", cb.name)
		cb.batchSize = size
	}

	if timeout < 0 {
		cb.errors = append(cb.errors, errors.New("invalid batch timeout provided"))
	} else {
		log.Infof(propSetMSG, "batchTimeout", cb.name)
		cb.batchTimeout = timeout
	}

	cb.batchTimeout = timeout
	return cb
}

// WithSyncCommit specifies that a consumer should commit offsets at the end of processing
// each message or batch of messages in a blocking synchronous operation.
func (cb *Builder) WithSyncCommit() *Builder {
	cb.commitSync = true
	return cb
}

// WithSaramaConfig specified a sarama consumer config. Use this to set consumer config or sarama level.
// Initialize a new sarama config:
//    c := sarama.NewConfig()
//
// Set consumer offset autocommit interval:
//    c.Consumer.Offsets.AutoCommit.Enable = false
//    c.Consumer.Offsets.AutoCommit.Interval = duration
//
// Set rebalance strategy
//    c.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky
//
// Set timeout
//    c.Net.DialTimeout = timeout
//
// Set Kafka version
//    c.Version = version
//
// Set offset
//    c.Consumer.Offsets.Initial = offset
//    c.Consumer.Offsets.Initial = sarama.OffsetOldest
//    c.Consumer.Offsets.Initial = sarama.OffsetNewest
//
// Check the sarama config documentation for more config options.
func (cb *Builder) WithSaramaConfig(cfg *sarama.Config) *Builder {
	cb.saramaConfig = cfg
	return cb
}

// Create constructs the Component applying
func (cb *Builder) Create() (*Component, error) {
	if len(cb.errors) > 0 {
		return nil, patronErrors.Aggregate(cb.errors...)
	}

	if cb.saramaConfig == nil {
		defaultSaramaCfg, err := defaultSaramaConfig(cb.name)
		if err != nil {
			return nil, err
		}

		cb.saramaConfig = defaultSaramaCfg
	}

	c := &Component{
		name:         cb.name,
		group:        cb.group,
		topics:       cb.topics,
		brokers:      cb.brokers,
		saramaConfig: cb.saramaConfig,
		proc:         cb.proc,
		failStrategy: cb.failStrategy,
		batchSize:    cb.batchSize,
		batchTimeout: cb.batchTimeout,
		retries:      cb.retries,
		retryWait:    cb.retryWait,
		commitSync:   cb.commitSync,
	}

	return c, nil
}

// Component is a kafka consumer implementation that processes messages in batch
type Component struct {
	name         string
	group        string
	topics       []string
	brokers      []string
	saramaConfig *sarama.Config
	proc         BatchProcessorFunc
	failStrategy FailStrategy
	batchSize    uint
	batchTimeout time.Duration
	retries      uint
	retryWait    time.Duration
	commitSync   bool
}

// Run starts the consumer processing loop messages.
func (c *Component) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	consumer := newConsumerHandler(ctx, cancel, c.name, c.group, c.proc, c.failStrategy, c.batchSize,
		c.batchTimeout, c.retries, c.retryWait, c.commitSync)
	client, err := sarama.NewConsumerGroup(c.brokers, c.group, c.saramaConfig)
	if err != nil {
		log.Errorf("error creating consumer group client for kafka consumer component: %v", err)
		return fmt.Errorf("error creating kafka consumer component: %w", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := client.Consume(ctx, c.topics, consumer); err != nil {
				log.Errorf("error from kafka consumer: %v", err)
			}

			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // wait for consumer to be set up
	log.Debug("kafka component: consumer ready")
	<-ctx.Done()
	log.Infof("kafka component terminating: context cancelled")
	wg.Wait()
	return client.Close()
}

// Consumer represents a sarama consumer group consumer
type consumerHandler struct {
	ctx    context.Context
	cancel context.CancelFunc

	name  string
	group string
	ready chan bool

	// buffer
	batchSize int
	ticker    *time.Ticker
	msgBuf    []*sarama.ConsumerMessage

	// retries
	retries   int
	retryWait time.Duration

	// lock to protect buffer operation
	mu sync.RWMutex

	// callback
	proc BatchProcessorFunc

	// failures strategy
	failStrategy FailStrategy

	// commit offsets after processing in a blocking synchronous operation
	commitSync bool
}

func newConsumerHandler(ctx context.Context, cancel context.CancelFunc, name, group string, processorFunc BatchProcessorFunc,
	fs FailStrategy, batchSize uint, batchTimeout time.Duration, retries uint, retryWait time.Duration,
	commitEveryBatch bool) *consumerHandler {

	return &consumerHandler{
		ctx:          ctx,
		cancel:       cancel,
		name:         name,
		group:        group,
		batchSize:    int(batchSize),
		ready:        make(chan bool),
		ticker:       time.NewTicker(batchTimeout),
		msgBuf:       make([]*sarama.ConsumerMessage, 0, batchSize),
		retries:      int(retries),
		retryWait:    retryWait,
		mu:           sync.RWMutex{},
		proc:         processorFunc,
		failStrategy: fs,
		commitSync:   commitEveryBatch,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *consumerHandler) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(c.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *consumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	// if the strategy is to exit on failure then we cancel the context to exit the whole component
	// and thus prevent the Consume loop from picking up the message again
	if c.failStrategy == ExitStrategy {
		c.cancel()
	}
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
			if err != nil {
				return err
			}
			c.mu.Unlock()
		case <-c.ctx.Done():
			if c.ctx.Err() != context.Canceled {
				log.Infof("closing consumer: %v", c.ctx.Err())
			}
			return nil
		}
	}
}

func (c *consumerHandler) flush(session sarama.ConsumerGroupSession) error {
	var err error

	if len(c.msgBuf) > 0 {
		ctxCh, cancel := c.getContextWithCorrelation()
		defer cancel()

		for i := 0; i <= c.retries; i++ {
			wrappedMessages := make([]MessageWrapper, 0, len(c.msgBuf))
			for _, msg := range c.msgBuf {
				messageStatusCountInc(messageProcessed, c.group, msg.Topic)
				wrappedMessages = append(wrappedMessages, &messageWrapper{msg: msg})
			}

			err = c.proc(ctxCh, wrappedMessages)
			if err == nil {
				break
			}

			if c.ctx.Err() == context.Canceled {
				return fmt.Errorf("context was cancelled after processing error: %w", err)
			}

			consumerErrorsInc(c.name)
			if c.retries > 0 {
				log.Errorf("failed run, retry %d/%d with %v wait: %v", i, c.retries, c.retryWait, err)
				time.Sleep(c.retryWait)
			}
		}

		if err != nil {
			for _, msg := range c.msgBuf {
				messageStatusCountInc(messageErrored, c.group, msg.Topic)
			}
			switch c.failStrategy {
			case ExitStrategy:
				log.Errorf("exhausted all retries but could not process message(s)")
				return err
			case SkipStrategy:
				log.Errorf("exhausted all retries but could not process message(s) so skipping")
			}
		}

		for _, m := range c.msgBuf {
			session.MarkMessage(m, "")
		}

		if c.commitSync {
			session.Commit()
		}

		c.msgBuf = make([]*sarama.ConsumerMessage, 0, c.batchSize)
	}

	return err
}

func (c *consumerHandler) getContextWithCorrelation() (context.Context, context.CancelFunc) {
	if len(c.msgBuf) == 1 {
		msg := c.msgBuf[0]
		corID := getCorrelationID(msg.Headers)

		_, ctxCh := trace.ConsumerSpan(c.ctx, trace.ComponentOpName(consumerComponent, msg.Topic),
			consumerComponent, corID, mapHeader(msg.Headers))
		ctxCh = correlation.ContextWithID(ctxCh, corID)
		ctxCh = log.WithContext(ctxCh, log.Sub(map[string]interface{}{correlation.ID: corID}))
		return ctxCh, func() {}
	}
	ctxCh, cancel := context.WithCancel(c.ctx)
	// generate a new correlation ID for batches larger than size 1
	batchCorrelationID := uuid.New().String()
	ctxCh = correlation.ContextWithID(ctxCh, batchCorrelationID)
	ctxCh = log.WithContext(ctxCh, log.Sub(map[string]interface{}{correlation.ID: batchCorrelationID}))
	return ctxCh, cancel
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

func mapHeader(hh []*sarama.RecordHeader) map[string]string {
	mp := make(map[string]string)
	for _, h := range hh {
		mp[string(h.Key)] = string(h.Value)
	}
	return mp
}
