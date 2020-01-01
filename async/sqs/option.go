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
func MaxMessages(maxMessages int64) OptionFunc {
	return func(f *Factory) error {
		if maxMessages <= 0 || maxMessages > 10 {
			return errors.New("max messages should be between 1 and 10")
		}
		f.maxMessages = maxMessages
		return nil
	}
}

// PollWaitSeconds sets the wait time for the long polling mechanism in seconds.
// Allowed values are between 0 and 20. 0 enables short polling.
func PollWaitSeconds(pollWaitSeconds int64) OptionFunc {
	return func(f *Factory) error {
		if pollWaitSeconds < 0 || pollWaitSeconds > 20 {
			return errors.New("poll wait seconds should be between 0 and 20")
		}
		f.pollWaitSeconds = pollWaitSeconds
		return nil
	}
}

// VisibilityTimeout sets the time a message is invisible after it has been requested.
// Allowed values are between 0 and and 12 hours in seconds.
func VisibilityTimeout(visibilityTimeout int64) OptionFunc {
	return func(f *Factory) error {
		if visibilityTimeout < 0 || visibilityTimeout > twelveHoursInSeconds {
			return fmt.Errorf("visibility timeout should be between 0 and %d seconds", twelveHoursInSeconds)
		}
		f.visibilityTimeout = visibilityTimeout
		return nil
	}
}

// Buffer sets the concurrency of the messages processing.
// 0 means wait for the previous messages to be processed.
func Buffer(buffer int) OptionFunc {
	return func(f *Factory) error {
		if buffer < 0 {
			return errors.New("buffer should be greater or equal to zero")
		}
		f.buffer = buffer
		return nil
	}
}

// QueueStatsInterval sets the interval at which we retrieve queue stats.
func QueueStatsInterval(interval time.Duration) OptionFunc {
	return func(f *Factory) error {
		if interval == 0 {
			return errors.New("queue stats interval should be a positive value")
		}
		f.statsInterval = interval
		return nil
	}
}
