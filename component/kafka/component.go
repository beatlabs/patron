package kafka

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/log"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/jcmturner/gokrb5.v7/config"
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
	cf           async.ConsumerFactory
	proc         BatchProcessorFunc
	failStrategy FailStrategy
	retries      uint
	retryWait    time.Duration
	errors       []error
}

// New initializes a new builder for a component with the given name
// by default the failStrategy will be NackExitStrategy.
func New(name string, cf async.ConsumerFactory, proc BatchProcessorFunc) *Builder {
	var errs []error
	if name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if cf == nil {
		errs = append(errs, errors.New("consumer is required"))
	}
	if proc == nil {
		errs = append(errs, errors.New("work processor is required"))
	}
	return &Builder{
		name:   name,
		cf:     cf,
		proc:   proc,
		errors: errs,
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

// WithRetries specifies the number of retries to be executed per failed batch
// default value is '0'.
func (cb *Builder) WithRetries(retries uint) *Builder {
	log.Infof(propSetMSG, "retries", cb.name)
	cb.retries = retries
	return cb
}

// WithRetryWait specifies the duration for the component to wait between retrying batches
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

// BatchConsumer is a kafka consumer implementation that processes messages in batch
type BatchConsumer struct {
	name    string
	group   string
	topics  []string
	brokers []string
}

// NewBatchConsumer initializes a new batch consumer
func NewBatchConsumer(ctx context.Context, cfg *config.Config) {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
	saramaCfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky
	batchProcessorFunc := func(messages []*sarama.ConsumerMessage) error {
		return nil
	}
	consumer := newConsumer(ctx, batchProcessorFunc, 100, 5*time.Second, false)

	ctx, cancel := context.WithCancel(context.Background())
	client, err := sarama.NewConsumerGroup([]string{cfg.KafkaBroker.Get()}, kafka2.ConsumerGroup(cfg.KafkaGroup.Get(), message.PassengerRelatedActionPositionTopic), saramaCfg)
	if err != nil {
		log.Errorf("Error creating consumer group client: %v", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := client.Consume(ctx, []string{message.PassengerRelatedActionPositionTopic}, consumer); err != nil {
				log.Errorf("error from consumer: %v", err)
			}

			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // wait for consumer to be set up
	log.Debug("consumer ready")

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
		log.Infof("terminating: context cancelled")
	case <-sigterm:
		log.Infof("terminating: via signal")
	}
	cancel()
	wg.Wait()
	if err = client.Close(); err != nil {
		log.Errorf("error closing client: %v", err)
	}
}

// Consumer represents a sarama consumer group consumer
type Consumer struct {
	ctx       context.Context
	batchSize int
	ready     chan bool

	// buffer
	ticker *time.Ticker
	msgBuf []*sarama.ConsumerMessage

	// lock to protect buffer operation
	mu sync.RWMutex

	// callback
	proc BatchProcessorFunc

	// commit every batch in a blocking synchronous operation
	commitEveryBatch bool
}

func newConsumer(ctx context.Context, processorFunc BatchProcessorFunc, batchSize int, timeout time.Duration, commitEveryBatch bool) *Consumer {
	return &Consumer{
		ctx:              ctx,
		batchSize:        batchSize,
		ready:            make(chan bool),
		ticker:           time.NewTicker(timeout),
		msgBuf:           make([]*sarama.ConsumerMessage, 0, batchSize),
		mu:               sync.RWMutex{},
		proc:             processorFunc,
		commitEveryBatch: commitEveryBatch,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if ok {
				log.Infof("message claimed: value = %s, timestamp = %v, topic = %s", string(msg.Value), msg.Timestamp, msg.Topic)
				err := consumer.insertMessage(session, msg)
				if err != nil {
					return err
				}
			} else {
				log.Debug("messages channel closed")
				return nil
			}
		case <-consumer.ticker.C:
			consumer.mu.Lock()
			err := consumer.flush(session)
			if err != nil {
				return err
			}
			consumer.mu.Unlock()
		case <-consumer.ctx.Done():
			if consumer.ctx.Err() != context.Canceled {
				log.Infof("closing consumer: %v", consumer.ctx.Err())
			}
			return nil
		}
	}
}

func (consumer *Consumer) flush(session sarama.ConsumerGroupSession) error {
	if len(consumer.msgBuf) > 0 {
		err := consumer.proc(consumer.msgBuf)
		if err != nil {
			return err
		}

		for _, m := range consumer.msgBuf {
			session.MarkMessage(m, "")
		}

		if consumer.commitEveryBatch {
			session.Commit()
		}

		consumer.msgBuf = make([]*sarama.ConsumerMessage, 0, consumer.batchSize)
	}

	return nil
}

func (consumer *Consumer) insertMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) error {
	consumer.mu.Lock()
	defer consumer.mu.Unlock()
	consumer.msgBuf = append(consumer.msgBuf, msg)
	if len(consumer.msgBuf) >= consumer.batchSize {
		return consumer.flush(session)
	}
	return nil
}

// BatchProcessorFunc definition of a batch async processor.
type BatchProcessorFunc func([]*sarama.ConsumerMessage) error
