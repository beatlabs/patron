package kafka

import (
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
)

// OptionFunc definition for configuring the async producer in a functional way.
type OptionFunc func(*AsyncProducer) error

// Version option for setting the version.
func Version(version string) OptionFunc {
	return func(ap *AsyncProducer) error {
		if version == "" {
			return errors.New("version is required")
		}
		v, err := sarama.ParseKafkaVersion(version)
		if err != nil {
			return errors.Wrap(err, "failed to parse kafka version")
		}
		ap.cfg.Version = v
		log.Infof("version %s set", version)
		return nil
	}
}

// Timeouts option for setting the timeouts.
func Timeouts(dial time.Duration) OptionFunc {
	return func(ap *AsyncProducer) error {
		if dial == 0 {
			return errors.New("dial timeout has to be positive")
		}
		ap.cfg.Net.DialTimeout = dial
		log.Infof("dial timeout %v set", dial)
		return nil
	}
}
