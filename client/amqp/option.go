package amqp

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// OptionFunc definition for configuring the publisher in a functional way.
type OptionFunc func(*Publisher) error

// WithConfig option for providing dial configuration.
func WithConfig(cfg amqp.Config) OptionFunc {
	return func(p *Publisher) error {
		p.cfg = &cfg
		return nil
	}
}
