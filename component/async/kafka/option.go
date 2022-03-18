package kafka

import (
	"errors"
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
)

// OptionFunc definition for configuring the consumer in a functional way.
type OptionFunc func(*ConsumerConfig) error

// Version for setting the Kafka version.
func Version(version string) OptionFunc {
	return func(c *ConsumerConfig) error {
		if version == "" {
			return errors.New("versions has to be provided")
		}

		v, err := sarama.ParseKafkaVersion(version)
		if err != nil {
			return fmt.Errorf("invalid kafka version provided: %w", err)
		}
		c.SaramaConfig.Version = v
		return nil
	}
}

// Buffer for adjusting the incoming messages buffer.
func Buffer(buf int) OptionFunc {
	return func(c *ConsumerConfig) error {
		if buf < 0 {
			return errors.New("buffer must greater or equal than 0")
		}
		c.Buffer = buf
		return nil
	}
}

// Timeout for adjusting the timeout of the connection.
func Timeout(timeout time.Duration) OptionFunc {
	return func(c *ConsumerConfig) error {
		c.SaramaConfig.Net.DialTimeout = timeout
		return nil
	}
}

// Start for adjusting the starting offset.
func Start(offset int64) OptionFunc {
	return func(c *ConsumerConfig) error {
		c.SaramaConfig.Consumer.Offsets.Initial = offset
		return nil
	}
}

// StartFromOldest for adjusting the starting offset to oldest.
func StartFromOldest() OptionFunc {
	return func(c *ConsumerConfig) error {
		c.SaramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
		return nil
	}
}

// StartFromNewest for adjusting the starting offset to newest.
func StartFromNewest() OptionFunc {
	return func(c *ConsumerConfig) error {
		c.SaramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
		return nil
	}
}

// Decoder for injecting a specific decoder implementation.
func Decoder(dec encoding.DecodeRawFunc) OptionFunc {
	return func(c *ConsumerConfig) error {
		if dec == nil {
			return errors.New("decoder is nil")
		}
		c.DecoderFunc = dec
		return nil
	}
}

// DecoderJSON for injecting json decoder.
func DecoderJSON() OptionFunc {
	return func(c *ConsumerConfig) error {
		c.DecoderFunc = json.DecodeRaw
		return nil
	}
}
