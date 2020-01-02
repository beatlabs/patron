package kafka

import (
	"time"

	"github.com/Shopify/sarama"

	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/errors"
)

// OptionFunc definition for configuring the consumer in a functional way.
type OptionFunc func(Consumer) error

// Version option for setting the Kafka version.
func Version(version string) OptionFunc {
	return func(c Consumer) error {
		if version == "" {
			return errors.New("versions has to be provided")
		}

		v, err := sarama.ParseKafkaVersion(version)
		if err != nil {
			return errors.Wrap(err, "invalid kafka version provided")
		}
		c.ConsumerConfig().SaramaConfig.Version = v
		return nil
	}
}

// Buffer option for adjusting the incoming messages buffer.
func Buffer(buf int) OptionFunc {
	return func(c Consumer) error {
		if buf < 0 {
			return errors.New("buffer must greater or equal than 0")
		}
		c.ConsumerConfig().Buffer = buf
		return nil
	}
}

// Timeout option for adjusting the timeout of the connection.
func Timeout(timeout time.Duration) OptionFunc {
	return func(c Consumer) error {
		c.ConsumerConfig().SaramaConfig.Net.DialTimeout = timeout
		return nil
	}
}

// Start option for adjusting the the starting offset
func Start(offset int64) OptionFunc {
	return func(c Consumer) error {
		c.ConsumerConfig().SaramaConfig.Consumer.Offsets.Initial = offset
		return nil
	}
}

// StartFromOldest option for adjusting the the starting offset to oldest
func StartFromOldest() OptionFunc {
	return func(c Consumer) error {
		c.ConsumerConfig().SaramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
		return nil
	}
}

// StartFromNewest option for adjusting the the starting offset to newest
func StartFromNewest() OptionFunc {
	return func(c Consumer) error {
		c.ConsumerConfig().SaramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
		return nil
	}
}

// Decoder option for injecting a specific decoder implementation
func Decoder(dec encoding.DecodeRawFunc) OptionFunc {
	return func(c Consumer) error {
		if dec == nil {
			return errors.New("decoder is nil")
		}
		c.ConsumerConfig().DecoderFunc = dec
		return nil
	}
}

// DecoderJSON option for injecting json decoder
func DecoderJSON() OptionFunc {
	return func(c Consumer) error {
		c.ConsumerConfig().DecoderFunc = json.DecodeRaw
		return nil
	}
}
