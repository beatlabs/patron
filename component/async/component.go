package async

import (
	"context"
	"errors"
	"fmt"
	"time"

	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/prometheus/client_golang/prometheus"
)

const propSetMSG = "property '%s' set for '%s'"

var consumerErrors *prometheus.CounterVec

func init() {
	consumerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: "async",
			Name:      "consumer_errors",
			Help:      "Consumer errors, classified by name and type",
		},
		[]string{"name"},
	)
	prometheus.MustRegister(consumerErrors)
}

func consumerErrorsInc(name string) {
	consumerErrors.WithLabelValues(name).Inc()
}

// Component implementation of a async component.
type Component struct {
	name         string
	proc         ProcessorFunc
	failStrategy FailStrategy
	cf           ConsumerFactory
	retries      int
	retryWait    time.Duration
	concurrency  int
	jobs         chan Message
	jobErr       chan error
}

// Builder gathers all required properties in order to construct a component.
type Builder struct {
	errors       []error
	name         string
	proc         ProcessorFunc
	failStrategy FailStrategy
	cf           ConsumerFactory
	retries      uint
	retryWait    time.Duration
	concurrency  uint
}

// New initializes a new builder for a component with the given name
// by default the failStrategy will be NackExitStrategy.
func New(name string, cf ConsumerFactory, proc ProcessorFunc) *Builder {
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
// default value is NackExitStrategy
// it will append an error to the builder if the strategy is not one of the pre-defined ones.
func (cb *Builder) WithFailureStrategy(fs FailStrategy) *Builder {
	if fs > AckStrategy || fs < NackExitStrategy {
		cb.errors = append(cb.errors, errors.New("invalid strategy provided"))
	} else {
		log.Debugf(propSetMSG, "failure strategy", cb.name)
		cb.failStrategy = fs
	}
	return cb
}

// WithRetries specifies the retry events number for the component
// default value is '0'.
func (cb *Builder) WithRetries(retries uint) *Builder {
	log.Debugf(propSetMSG, "retries", cb.name)
	cb.retries = retries
	return cb
}

// WithConcurrency specifies the number of worker goroutines for processing messages in parallel
// default value is '1'
// do NOT enable concurrency value for in-order consumers, such as Kafka or FIFO SQS.
func (cb *Builder) WithConcurrency(concurrency uint) *Builder {
	log.Debugf(propSetMSG, "concurrency", cb.name)
	cb.concurrency = concurrency
	return cb
}

// WithRetryWait specifies the duration for the component to wait between retries
// default value is '0'
// it will append an error to the builder if the value is smaller than '0'.
func (cb *Builder) WithRetryWait(retryWait time.Duration) *Builder {
	if retryWait < 0 {
		cb.errors = append(cb.errors, errors.New("invalid retry wait provided"))
	} else {
		log.Debugf(propSetMSG, "retryWait", cb.name)
		cb.retryWait = retryWait
	}
	return cb
}

// Create constructs the Component applying.
func (cb *Builder) Create() (*Component, error) {
	if len(cb.errors) > 0 {
		return nil, patronErrors.Aggregate(cb.errors...)
	}

	c := &Component{
		name:         cb.name,
		proc:         cb.proc,
		cf:           cb.cf,
		failStrategy: cb.failStrategy,
		retries:      int(cb.retries),
		retryWait:    cb.retryWait,
		concurrency:  int(cb.concurrency),
		jobs:         make(chan Message),
		jobErr:       make(chan error),
	}

	if cb.concurrency > 1 {
		for w := 1; w <= c.concurrency; w++ {
			go c.worker()
		}
	}

	return c, nil
}

// Run starts the consumer processing loop messages.
func (c *Component) Run(ctx context.Context) error {
	var err error

	for i := 0; i <= c.retries; i++ {
		err = c.processing(ctx)
		if err == nil {
			return nil
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			break
		}
		consumerErrorsInc(c.name)
		if c.retries > 0 {
			log.Errorf("failed run, retry %d/%d with %v wait: %v", i, c.retries, c.retryWait, err)
			time.Sleep(c.retryWait)
		}
	}

	close(c.jobs)
	return err
}

func (c *Component) processing(ctx context.Context) error {
	cns, err := c.cf.Create()
	if c.concurrency > 1 && !cns.OutOfOrder() {
		return fmt.Errorf("async component creation: cannot create in-order component with concurrency > 1")
	}
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}
	defer func() {
		err := cns.Close()
		if err != nil {
			log.Warnf("failed to close consumer: %v", err)
		}
	}()

	chMsg, chErr, err := cns.Consume(ctx)
	if err != nil {
		return fmt.Errorf("failed to get consumer channels: %w", err)
	}

	for {
		select {
		case msg := <-chMsg:
			log.FromContext(msg.Context()).Debug("consumer received a new message")
			err := c.dispatchMessage(msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				log.Warnf("closing consumer: %v", ctx.Err())
			}
			return cns.Close()
		case err := <-chErr:
			return fmt.Errorf("an error occurred during message consumption: %w", err)
		case err := <-c.jobErr:
			return fmt.Errorf("an error occurred during concurrent message consumption: %w", err)
		}
	}
}

func (c *Component) dispatchMessage(msg Message) error {
	if c.concurrency > 1 {
		c.jobs <- msg
		return nil
	}
	return c.processMessage(msg)
}

func (c *Component) processMessage(msg Message) error {
	err := c.proc(msg)
	if err != nil {
		return c.executeFailureStrategy(msg, err)
	}
	return msg.Ack()
}

func (c *Component) worker() {
	for msg := range c.jobs {
		err := c.processMessage(msg)
		if err != nil {
			c.jobErr <- err
		}
	}
}

var errInvalidFS = errors.New("invalid failure strategy")

func (c *Component) executeFailureStrategy(msg Message, err error) error {
	log.FromContext(msg.Context()).Errorf("failed to process message, failure strategy executed: %v", err)
	switch c.failStrategy {
	case NackExitStrategy:
		nackErr := msg.Nack()
		if nackErr != nil {
			return patronErrors.Aggregate(err, fmt.Errorf("failed to NACK message: %w", nackErr))
		}
		return err
	case NackStrategy:
		err := msg.Nack()
		if err != nil {
			return fmt.Errorf("nack failed when executing failure strategy: %w", err)
		}
	case AckStrategy:
		err := msg.Ack()
		if err != nil {
			return fmt.Errorf("ack failed when executing failure strategy: %w", err)
		}
	default:
		return errInvalidFS
	}
	return nil
}
