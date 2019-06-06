package amqp

import (
	"net"
	"time"

	"github.com/beatlabs/patron/errors"
	"github.com/streadway/amqp"
)

// OptionFunc definition for configuring the consumer in a functional way.
type OptionFunc func(*consumer) error

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
		c.cfg = amqp.Config{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, timeout)
			},
		}
		return nil
	}
}

// Requeue option for adjusting the requeue policy of a message.
func Requeue(requeue bool) OptionFunc {
	return func(c *consumer) error {
		c.requeue = requeue
		return nil
	}
}

// Bindings option for providing custom exchange-queue bindings.
func Bindings(bindings ...string) OptionFunc {
	return func(c *consumer) error {
		if len(bindings) == 0 {
			return errors.New("Provided bindings cannot be empty")
		}

		c.bindings = bindings
		return nil
	}
}
