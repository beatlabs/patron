package async

import (
	"context"
	"time"

	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/prometheus/client_golang/prometheus"
)

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
		retries:      0,
		retryWait:    0,
	}

	for _, o := range oo {
		err := o(c)
		if err != nil {
			return nil, err
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
		if ctx.Err() == context.Canceled {
			break
		}
		consumerErrorsInc(c.name)
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
	defer func() {
		err = cns.Close()
		if err != nil {
			log.Warnf("failed to close consumer: %v", err)
		}
	}()

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
				c.processMessage(msg, failCh)
			case errMsg := <-chErr:
				failCh <- errors.Wrap(errMsg, "an error occurred during message consumption")
				return
			}
		}
	}()
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
