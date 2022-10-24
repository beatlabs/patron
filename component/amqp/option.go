package amqp

import (
	"errors"
	"time"

	"github.com/streadway/amqp"
)

// OptionFunc definition for configuring the component in a functional way.
type OptionFunc func(*Component) error

// WithBatching option for setting up batching.
// Allowed values for count is > 1 and timeout > 0.
func WithBatching(count uint, timeout time.Duration) OptionFunc {
	return func(c *Component) error {
		if count == 0 || count == 1 {
			return errors.New("count should be larger than 1 message")
		}
		if timeout <= 0 {
			return errors.New("timeout should be a positive number")
		}

		c.batchCfg.count = count
		c.batchCfg.timeout = timeout
		return nil
	}
}

// WithRetry option for setting up retries.
func WithRetry(count uint, delay time.Duration) OptionFunc {
	return func(c *Component) error {
		c.retryCfg.count = count
		c.retryCfg.delay = delay
		return nil
	}
}

// WithConfig option for setting AMQP configuration.
func WithConfig(cfg amqp.Config) OptionFunc {
	return func(c *Component) error {
		c.cfg = cfg
		return nil
	}
}

// WithStatsInterval option for setting the interval to retrieve statistics.
func WithStatsInterval(interval time.Duration) OptionFunc {
	return func(c *Component) error {
		if interval <= 0 {
			return errors.New("stats interval should be a positive number")
		}
		c.statsCfg.interval = interval
		return nil
	}
}

// WithRequeue option for adjusting the requeue policy of a message.
func WithRequeue(requeue bool) OptionFunc {
	return func(c *Component) error {
		c.queueCfg.requeue = requeue
		return nil
	}
}
