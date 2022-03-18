package sqs

import (
	"errors"
	"fmt"
	"time"
)

const twelveHoursInSeconds = 43200

// OptionFunc definition for configuring the consumer in a functional way.
type OptionFunc func(*Factory) error

// MaxMessages option for setting the max number of messages fetched.
// Allowed values are between 1 and 10.
// If this option is unused, it defaults to 3.
// If messages can be processed very quickly, maxing out this value is fine, otherwise having a high value is risky as it might trigger the visibility timeout.
// Having a value too small isn't recommended either, as it increases the number of SQS API requests, thus AWS costs.
func MaxMessages(maxMessages int64) OptionFunc {
	return func(f *Factory) error {
		if maxMessages <= 0 || maxMessages > 10 {
			return errors.New("max messages should be between 1 and 10")
		}
		f.maxMessages = &maxMessages
		return nil
	}
}

// PollWaitSeconds sets the wait time for the long polling mechanism in seconds.
// Allowed values are between 0 and 20. 0 enables short polling.
// If this option is unused, it defaults to the queue's default poll settings.
func PollWaitSeconds(pollWaitSeconds int64) OptionFunc {
	return func(f *Factory) error {
		if pollWaitSeconds < 0 || pollWaitSeconds > 20 {
			return errors.New("poll wait seconds should be between 0 and 20")
		}
		f.pollWaitSeconds = &pollWaitSeconds
		return nil
	}
}

// VisibilityTimeout sets the time a message is invisible after it has been requested.
// This is a built-in resiliency mechanism so that, should the consumer fail to acknowledge the message within such timeout,
// it will become visible again and thus available for retries.
// Allowed values are between 0 and 12 hours in seconds.
// If this option is unused, it defaults to the queue's default visibility settings.
func VisibilityTimeout(visibilityTimeout int64) OptionFunc {
	return func(f *Factory) error {
		if visibilityTimeout < 0 || visibilityTimeout > twelveHoursInSeconds {
			return fmt.Errorf("visibility timeout should be between 0 and %d seconds", twelveHoursInSeconds)
		}
		f.visibilityTimeout = &visibilityTimeout
		return nil
	}
}

// QueueStatsInterval sets the interval at which we retrieve queue stats.
// If this option is unused, it defaults to 20 seconds.
func QueueStatsInterval(interval time.Duration) OptionFunc {
	return func(f *Factory) error {
		if interval == 0 {
			return errors.New("queue stats interval should be a positive value")
		}
		f.statsInterval = interval
		return nil
	}
}
