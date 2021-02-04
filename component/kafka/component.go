package kafka

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/internal/validation"
	"github.com/beatlabs/patron/log"
	"github.com/prometheus/client_golang/prometheus"
)

const propSetMSG = "property '%s' set for '%s'"

var consumerErrors *prometheus.CounterVec

func init() {
	consumerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "kafka",
			Name:      "consumer_errors",
			Help:      "Consumer errors, classified by consumer name",
		},
		[]string{"name"},
	)
	prometheus.MustRegister(consumerErrors)
}

func consumerErrorsInc(name string) {
	consumerErrors.WithLabelValues(name).Inc()
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
	batchSize    int
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
func (cb *Builder) WithBatching(size int, timeout time.Duration) *Builder {
	if size <= 0 {
		cb.errors = append(cb.errors, errors.New("invalid batch size provided"))
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
	batchSize    int
	batchTimeout time.Duration
	retries      uint
	retryWait    time.Duration
	commitSync   bool
}

// Run starts the consumer processing loop messages.
func (c *Component) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	consumer := newConsumerHandler(ctx, c.name, c.proc, c.failStrategy, c.batchSize, c.batchTimeout, c.retries, c.retryWait, c.commitSync)
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
	log.Debug("kafka consumer component: consumer ready")

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
		log.Infof("kafka consumer component terminating: context cancelled")
	case <-sigterm:
		log.Infof("kafka consumer component terminating: via signal")
	}
	wg.Wait()
	return client.Close()
}

// Consumer represents a sarama consumer group consumer
type consumerHandler struct {
	ctx       context.Context
	name      string
	batchSize int
	ready     chan bool

	// buffer
	ticker *time.Ticker
	msgBuf []*sarama.ConsumerMessage

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

func newConsumerHandler(ctx context.Context, name string, processorFunc BatchProcessorFunc, fs FailStrategy,
	batchSize int, batchTimeout time.Duration, retries uint, retryWait time.Duration,
	commitEveryBatch bool) *consumerHandler {

	return &consumerHandler{
		ctx:        ctx,
		name:       name,
		batchSize:  batchSize,
		ready:      make(chan bool),
		ticker:     time.NewTicker(batchTimeout),
		msgBuf:     make([]*sarama.ConsumerMessage, 0, batchSize),
		retries:    int(retries),
		retryWait:  retryWait,
		mu:         sync.RWMutex{},
		proc:       processorFunc,
		commitSync: commitEveryBatch,
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
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (c *consumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if ok {
				log.Infof("message claimed: value = %s, timestamp = %v, topic = %s", string(msg.Value), msg.Timestamp, msg.Topic)
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
		for i := 0; i <= c.retries; i++ {
			err = c.proc(c.msgBuf)
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

func (c *consumerHandler) insertMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgBuf = append(c.msgBuf, msg)
	if len(c.msgBuf) >= c.batchSize {
		return c.flush(session)
	}
	return nil
}

// BatchProcessorFunc definition of a batch async processor.
type BatchProcessorFunc func([]*sarama.ConsumerMessage) error
