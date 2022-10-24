package amqp

import (
	"errors"
	"net"
	"time"

	"github.com/streadway/amqp"
)

// OptionFunc definition for configuring the consumer in a functional way.
type OptionFunc func(*consumer) error

// WithBuffer option for adjusting the incoming messages buffer.
func WithBuffer(buf int) OptionFunc {
	return func(c *consumer) error {
		if buf < 0 {
			return errors.New("buffer must greater or equal than 0")
		}
		c.buffer = buf
		return nil
	}
}

// WithTimeout option for adjusting the timeout of the connection.
func WithTimeout(timeout time.Duration) OptionFunc {
	return func(c *consumer) error {
		c.cfg = amqp.Config{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, timeout)
			},
		}
		return nil
	}
}

// WithRequeue option for adjusting the requeue policy of a message.
func WithRequeue(requeue bool) OptionFunc {
	return func(c *consumer) error {
		c.requeue = requeue
		return nil
	}
}

// WithBindings option for providing custom exchange-queue bindings.
func WithBindings(bindings ...string) OptionFunc {
	return func(c *consumer) error {
		if len(bindings) == 0 {
			return errors.New("provided bindings cannot be empty")
		}

		c.bindings = bindings
		return nil
	}
}
