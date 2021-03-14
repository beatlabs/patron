package amqp

import (
	"errors"
	"time"

	"github.com/streadway/amqp"
)

// OptionFunc definition for configuring the component in a functional way.
type OptionFunc func(*Component) error

// Batching option for setting up batching.
// Allowed values for count is > 1 and timeout > 0.
func Batching(count uint, timeout time.Duration) OptionFunc {
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

// Retry option for setting up retries.
func Retry(count uint, delay time.Duration) OptionFunc {
	return func(c *Component) error {
		c.retryCfg.count = count
		c.retryCfg.delay = delay
		return nil
	}
}

// Config option for setting AMQP configuration.
func Config(cfg amqp.Config) OptionFunc {
	return func(c *Component) error {
		c.cfg = cfg
		return nil
	}
}

// StatsInterval option for setting the interval to retrieve statistics.
func StatsInterval(interval time.Duration) OptionFunc {
	return func(c *Component) error {
		if interval <= 0 {
			return errors.New("stats interval should be a positive number")
		}
		c.statsCfg.interval = interval
		return nil
	}
}

// Requeue option for adjusting the requeue policy of a message.
func Requeue(requeue bool) OptionFunc {
	return func(c *Component) error {
		c.queueCfg.requeue = requeue
		return nil
	}
}
