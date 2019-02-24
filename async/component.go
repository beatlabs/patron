package async

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/thebeatapp/patron/errors"
	"github.com/thebeatapp/patron/log"
	"github.com/thebeatapp/patron/metric"
)

var consumerErrors *prometheus.CounterVec

// Component implementation of a async component.
type Component struct {
	name         string
	proc         ProcessorFunc
	failStrategy FailStrategy
	readiness    func()
	cf           ConsumerFactory
	retries      int
	retryWait    time.Duration
	info         map[string]interface{}
}

// New returns a new async component. The default behavior is to return a error of failure.
// Use options to change the default behavior.
func New(name string, p ProcessorFunc, cf ConsumerFactory, oo ...OptionFunc) (*Component, error) {

	if name == "" {
		return nil, errors.New("name is required")
	}

	if p == nil {
		return nil, errors.New("work processor is required")
	}

	if cf == nil {
		return nil, errors.New("consumer is required")
	}

	c := &Component{
		name:         name,
		proc:         p,
		cf:           cf,
		failStrategy: NackExitStrategy,
		readiness:    func() {},
		retries:      0,
		retryWait:    0,
		info:         make(map[string]interface{}),
	}

	for _, o := range oo {
		err := o(c)
		if err != nil {
			return nil, err
		}
	}

	c.setupInfo()
	err := setupMetrics()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Info return information of the component.
func (c *Component) Info() map[string]interface{} {
	return c.info
}

// Run starts the consumer processing loop messages.
func (c *Component) Run(ctx context.Context) error {

	var err error

	for i := 0; i <= c.retries; i++ {
		err = c.processing(ctx)
		if err == nil {
			return nil
		}
		if ctx.Err() == context.Canceled {
			break
		}
		c.consumerErrorsInc()
		if c.retries > 0 {
			log.Errorf("failed run, retry %d/%d with %v wait: %v", i, c.retries, c.retryWait, err)
			time.Sleep(c.retryWait)
		}
	}

	return err
}

func (c *Component) processing(ctx context.Context) error {

	cns, err := c.cf.Create()
	if err != nil {
		return errors.Wrap(err, "failed to create consumer")
	}
	defer cns.Close()
	c.info["consumer"] = cns.Info()

	chMsg, chErr, err := cns.Consume(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get consumer channels")
	}

	failCh := make(chan error)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info("closing consumer")
				failCh <- cns.Close()
			case msg := <-chMsg:
				log.Debug("New message from consumer arrived")
				go c.processMessage(msg, failCh)
			case errMsg := <-chErr:
				failCh <- errors.Wrap(errMsg, "an error occurred during message consumption")
				return
			}
		}
	}()

	c.readiness()

	return <-failCh
}

func (c *Component) processMessage(msg Message, ch chan error) {
	err := c.proc(msg)
	if err != nil {
		err := c.executeFailureStrategy(msg, err)
		if err != nil {
			ch <- err
		}
		return
	}
	if err := msg.Ack(); err != nil {
		ch <- err
	}
}

func (c *Component) executeFailureStrategy(msg Message, err error) error {
	log.Errorf("failed to process message, failure strategy executed: %v", err)
	switch c.failStrategy {
	case NackExitStrategy:
		return errors.Aggregate(err, errors.Wrap(msg.Nack(), "failed to NACK message"))
	case NackStrategy:
		err := msg.Nack()
		if err != nil {
			return errors.Wrap(err, "nack failed when executing failure strategy")
		}
	case AckStrategy:
		err := msg.Ack()
		if err != nil {
			return errors.Wrap(err, "ack failed when executing failure strategy")
		}
	default:
		return errors.New("invalid failure strategy")
	}
	return nil
}

func (c *Component) setupInfo() {
	c.info["type"] = "async"
	c.info["fail-strategy"] = c.failStrategy.String()
	c.info["consumer-retries"] = c.retries
	c.info["consumer-timeout"] = c.retryWait.String()
}

func setupMetrics() error {
	var err error
	consumerErrors, err = metric.NewCounter(
		"async_component",
		"consumer_errors",
		"Consumer errors, classified by name and type",
		"name",
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *Component) consumerErrorsInc() {
	consumerErrors.WithLabelValues(c.name).Inc()
}
