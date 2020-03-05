package kafka

import (
	"errors"
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/log"
)

// OptionFunc definition for configuring the async producer in a functional way.
type OptionFunc func(producer *SyncProducer) error

// RequiredAcks is used in Produce Requests to tell the broker how many replica acknowledgements
// it must see before responding.
type RequiredAcks int16

const (
	// NoResponse doesn't send any response, the TCP ACK is all you get.
	NoResponse RequiredAcks = 0
	// WaitForLocal waits for only the local commit to succeed before responding.
	WaitForLocal RequiredAcks = 1
	// WaitForAll waits for all in-sync replicas to commit before responding.
	WaitForAll RequiredAcks = -1
)

// Version option for setting the version. Producing headers, which are used for
// tracing propagation requires at least version 0.11 of kafka. Thus, this
// component supports only versions greater than 11.0.
func Version(version string) OptionFunc {
	return func(sp *SyncProducer) error {
		if version == "" {
			return errors.New("version is required")
		}
		v, err := sarama.ParseKafkaVersion(version)
		if err != nil {
			return fmt.Errorf("failed to parse kafka version: %w", err)
		}
		if !v.IsAtLeast(sarama.V0_11_0_0) {
			return errors.New("version lower than 11.0.0 is not supported")
		}

		sp.cfg.Version = v
		log.Infof("version %s set", version)
		return nil
	}
}

// Timeouts option for setting the timeouts.
func Timeouts(dial time.Duration) OptionFunc {
	return func(ap *SyncProducer) error {
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
func RequiredAcksPolicy(ack RequiredAcks) OptionFunc {
	return func(sp *SyncProducer) error {
		sp.cfg.Producer.RequiredAcks = sarama.RequiredAcks(ack)
		return nil
	}
}
