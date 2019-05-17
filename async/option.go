package async

import (
	"time"

	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
)

// FailStrategy type definition.
type FailStrategy int

const (
	// NackExitStrategy does not acknowledge the message and exits the application on error.
	NackExitStrategy FailStrategy = 0
	// NackStrategy does not acknowledge the message, leaving it for reprocessing, and continues.
	NackStrategy FailStrategy = 1
	// AckStrategy acknowledges message and continues.
	AckStrategy FailStrategy = 2
)

func (fs FailStrategy) String() string {
	switch fs {
	case NackExitStrategy:
		return "NackExitStrategy"
	case NackStrategy:
		return "NackStrategy"
	case AckStrategy:
		return "AckStrategy"
	default:
		return "N/A"
	}
}

// OptionFunc definition for configuring the component in a functional way.
type OptionFunc func(*Component) error

// FailureStrategy option for setting the strategy of handling failures in the async component.
func FailureStrategy(fs FailStrategy) OptionFunc {
	return func(c *Component) error {
		if fs > AckStrategy || fs < NackExitStrategy {
			return errors.New("invalid strategy provided")
		}
		c.failStrategy = fs
		log.Info("failure strategy set")
		return nil
	}
}

// ConsumerRetry set's the parameters for the retry policy of the consumer.
func ConsumerRetry(retries int, retryWait time.Duration) OptionFunc {
	return func(c *Component) error {
		if retries < 0 {
			return errors.New("invalid retries provided")
		}

		if retryWait < 0 {
			return errors.New("invalid retry wait provided")
		}

		c.retries = retries
		c.retryWait = retryWait
		log.Info("consumer retry set")
		return nil
	}
}
