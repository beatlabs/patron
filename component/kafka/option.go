package kafka

import (
	"errors"
	"time"

	"github.com/beatlabs/patron/log"

	"github.com/Shopify/sarama"
)

// OptionFunc definition for configuring the component in a functional way.
type OptionFunc func(*Component) error

// FailureStrategy sets the strategy to follow for the component when it encounters an error.
func FailureStrategy(fs FailStrategy) OptionFunc {
	return func(c *Component) error {
		if fs > SkipStrategy || fs < ExitStrategy {
			return errors.New("invalid failure strategy provided")
		}
		c.failStrategy = fs
		return nil
	}
}

// Retries sets the error retries of the component.
func Retries(count uint) OptionFunc {
	return func(c *Component) error {
		c.retries = count
		return nil
	}
}

// RetryWait sets the wait period for the component retry.
func RetryWait(interval time.Duration) OptionFunc {
	return func(c *Component) error {
		if interval <= 0 {
			return errors.New("retry wait time should be a positive number")
		}
		c.retryWait = interval
		return nil
	}
}

// BatchSize sets the message batch size the component should process at once.
func BatchSize(size uint) OptionFunc {
	return func(c *Component) error {
		if size == 0 {
			return errors.New("zero batch size provided")
		}
		c.batchSize = size
		return nil
	}
}

// BatchTimeout sets the message batch timeout. If the desired batch size is not reached and if the timeout elapses
// without new messages coming in, the messages in the buffer would get processed as a batch.
func BatchTimeout(timeout time.Duration) OptionFunc {
	return func(c *Component) error {
		if timeout < 0 {
			return errors.New("batch timeout should greater than or equal to zero")
		}
		c.batchTimeout = timeout
		return nil
	}
}

// SaramaConfig specifies a sarama consumer config. Use this to set consumer config on sarama level.
// Check the sarama config documentation for more config options.
func SaramaConfig(cfg *sarama.Config) OptionFunc {
	return func(c *Component) error {
		if cfg == nil {
			return errors.New("nil sarama configuration provided")
		}
		c.saramaConfig = cfg
		return nil
	}
}

// CommitSync instructs the consumer to commit offsets in a blocking operation after processing every batch of messages
func CommitSync() OptionFunc {
	return func(c *Component) error {
		if c.saramaConfig != nil && c.saramaConfig.Consumer.Offsets.AutoCommit.Enable {
			// redundant commits warning
			log.Warn("consumer is set to commit offsets after processing each batch and auto-commit is enabled")
		}
		c.commitSync = true
		return nil
	}
}
