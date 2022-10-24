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

// WithVersion for setting the Kafka version.
func WithVersion(version string) OptionFunc {
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

// WithBuffer for adjusting the incoming messages buffer.
func WithBuffer(buf int) OptionFunc {
	return func(c *ConsumerConfig) error {
		if buf < 0 {
			return errors.New("buffer must greater or equal than 0")
		}
		c.Buffer = buf
		return nil
	}
}

// WithTimeout for adjusting the timeout of the connection.
func WithTimeout(timeout time.Duration) OptionFunc {
	return func(c *ConsumerConfig) error {
		c.SaramaConfig.Net.DialTimeout = timeout
		return nil
	}
}

// WithStart for adjusting the starting offset.
func WithStart(offset int64) OptionFunc {
	return func(c *ConsumerConfig) error {
		c.SaramaConfig.Consumer.Offsets.Initial = offset
		return nil
	}
}

// WithStartFromOldest for adjusting the starting offset to oldest.
func WithStartFromOldest() OptionFunc {
	return func(c *ConsumerConfig) error {
		c.SaramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
		return nil
	}
}

// WithStartFromNewest for adjusting the starting offset to newest.
func WithStartFromNewest() OptionFunc {
	return func(c *ConsumerConfig) error {
		c.SaramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
		return nil
	}
}

// WithDecoder for injecting a specific decoder implementation.
func WithDecoder(dec encoding.DecodeRawFunc) OptionFunc {
	return func(c *ConsumerConfig) error {
		if dec == nil {
			return errors.New("decoder is nil")
		}
		c.DecoderFunc = dec
		return nil
	}
}

// WithDecoderJSON for injecting json decoder.
func WithDecoderJSON() OptionFunc {
	return func(c *ConsumerConfig) error {
		c.DecoderFunc = json.DecodeRaw
		return nil
	}
}
