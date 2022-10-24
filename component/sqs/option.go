package sqs

import (
	"errors"
	"fmt"
	"time"
)

const twelveHoursInSeconds = 43200

// OptionFunc definition for configuring the component in a functional way.
type OptionFunc func(*Component) error

// WithMaxMessages option for setting the max number of messages fetched.
// Allowed values are between 1 and 10.
// If messages can be processed very quickly, maxing out this value is fine, otherwise having a high value is risky as it might trigger the visibility timeout.
// Having a value too small isn't recommended either, as it increases the number of SQS API requests, thus AWS costs.
func WithMaxMessages(maxMessages int32) OptionFunc {
	return func(c *Component) error {
		if maxMessages <= 0 || maxMessages > 10 {
			return errors.New("max messages should be between 1 and 10")
		}
		c.cfg.maxMessages = maxMessages
		return nil
	}
}

// WithPollWaitSeconds sets the wait time for the long polling mechanism in seconds.
// Allowed values are between 0 and 20. 0 enables short polling.
func WithPollWaitSeconds(pollWaitSeconds int32) OptionFunc {
	return func(c *Component) error {
		if pollWaitSeconds < 0 || pollWaitSeconds > 20 {
			return errors.New("poll wait seconds should be between 0 and 20")
		}
		c.cfg.pollWaitSeconds = pollWaitSeconds
		return nil
	}
}

// WithVisibilityTimeout sets the time a message is invisible after it has been requested.
// This is a built-in resiliency mechanism so that, should the consumer fail to acknowledge the message within such timeout,
// it will become visible again and thus available for retries.
// Allowed values are between 0 and 12 hours in seconds.
func WithVisibilityTimeout(visibilityTimeout int32) OptionFunc {
	return func(c *Component) error {
		if visibilityTimeout < 0 || visibilityTimeout > twelveHoursInSeconds {
			return fmt.Errorf("visibility timeout should be between 0 and %d seconds", twelveHoursInSeconds)
		}
		c.cfg.visibilityTimeout = visibilityTimeout
		return nil
	}
}

// WithQueueStatsInterval sets the interval at which we retrieve AWS SQS stats.
func WithQueueStatsInterval(interval time.Duration) OptionFunc {
	return func(c *Component) error {
		if interval == 0 {
			return errors.New("sqsAPI stats interval should be a positive value")
		}
		c.stats.interval = interval
		return nil
	}
}

// WithRetries sets the error retries of the component.
func WithRetries(count uint) OptionFunc {
	return func(c *Component) error {
		c.retry.count = count
		return nil
	}
}

// WithRetryWait sets the wait period for the component retry.
func WithRetryWait(interval time.Duration) OptionFunc {
	return func(c *Component) error {
		if interval <= 0 {
			return errors.New("retry wait time should be a positive number")
		}
		c.retry.wait = interval
		return nil
	}
}

// WithQueueOwner sets the AWS queue owner ID for an SQS queue.
func WithQueueOwner(queueOwner string) OptionFunc {
	return func(c *Component) error {
		if queueOwner == "" {
			return errors.New("queue owner should not be empty")
		}
		c.queueOwner = queueOwner
		return nil
	}
}
