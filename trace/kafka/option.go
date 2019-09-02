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

// RequiredAcksPolicy option for adjusting how many replica acknowledgements
// broker must see before responding.
// 0 doesn't send any response, the TCP ACK is all you get.
// 1 waits for only the local commit to succeed before responding.
// -1  waits for all in-sync replicas to commit before responding.
func RequiredAcksPolicy(ack int) OptionFunc {
	return func(ap *AsyncProducer) error {
		switch ack {
		case -1:
		case 0:
		case 1:
			ap.cfg.Producer.RequiredAcks = sarama.RequiredAcks(ack)
		default:
			return errors.New("invalid required acks policy provided")
		}
		return nil
	}
}
