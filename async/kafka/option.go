package kafka

import (
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/errors"
)

// OptionFunc definition for configuring the consumer in a functional way.
type OptionFunc func(*consumer) error

// Version option for setting the Kafka version.
func Version(version string) OptionFunc {
	return func(c *consumer) error {
		if version == "" {
			return errors.New("versions has to be provided")
		}

		v, err := sarama.ParseKafkaVersion(version)
		if err != nil {
			return errors.Wrap(err, "invalid kafka version provided")
		}

		c.cfg.Version = v
		return nil
	}
}

// Buffer option for adjusting the incoming messages buffer.
func Buffer(buf int) OptionFunc {
	return func(c *consumer) error {
		if buf < 0 {
			return errors.New("buffer must greater or equal than 0")
		}
		c.buffer = buf
		return nil
	}
}

// Timeout option for adjusting the timeout of the connection.
func Timeout(timeout time.Duration) OptionFunc {
	return func(c *consumer) error {
		c.cfg.Net.DialTimeout = timeout
		return nil
	}
}

// Start option for adjusting the the starting offset
func Start(offset int64) OptionFunc {
	return func(c *consumer) error {
		c.cfg.Consumer.Offsets.Initial = offset
		return nil
	}
}

// RequiredAcksPolicy option for adjusting how many replica acknowledgements
// broker must see before responding.
// 0 doesn't send any response, the TCP ACK is all you get.
// 1 waits for only the local commit to succeed before responding.
// -1  waits for all in-sync replicas to commit before responding.
func RequiredAcksPolicy(ack int16) OptionFunc {
	return func(c *consumer) error {
		if ack < -1 {
			return errors.New("required acks policy must be greater or equal than -1")
		}
		if ack > 1 {
			return errors.New("required acks policy must be lesser or equal than 1")
		}
		c.cfg.Producer.RequiredAcks = sarama.RequiredAcks(ack)
		return nil
	}
}
