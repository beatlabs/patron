package amqp

import (
	"net"
	"time"

	"github.com/beatlabs/patron/errors"
	"github.com/streadway/amqp"
)

// OptionFunc definition for configuring the publisher in a functional way.
type OptionFunc func(*TracedPublisher) error

// Timeout option for adjusting the timeout of the connection.
func Timeout(timeout time.Duration) OptionFunc {
	return func(tp *TracedPublisher) error {
		if timeout <= 0 {
			return errors.New("timeout must be positive")
		}
		tp.cfg = amqp.Config{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, timeout)
			},
		}
		return nil
	}
}
