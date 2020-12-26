package v2

import (
	"github.com/streadway/amqp"
)

// OptionFunc definition for configuring the publisher in a functional way.
type OptionFunc func(*Publisher) error

// Config option for providing dial configuration.
func Config(cfg amqp.Config) OptionFunc {
	return func(p *Publisher) error {
		p.cfg = &cfg
		return nil
	}
}
