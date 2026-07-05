package kafka

import "github.com/twmb/franz-go/pkg/kgo"

// OptionFunc configures a Kafka producer.
type OptionFunc func(*config)

type config struct {
	kgoOpts []kgo.Opt
}

// WithKafkaOptions adds franz-go options to the underlying Kafka client.
func WithKafkaOptions(opts ...kgo.Opt) OptionFunc {
	return func(cfg *config) {
		cfg.kgoOpts = append(cfg.kgoOpts, opts...)
	}
}
