package mqtt

import (
	"errors"
)

// OptionFunc definition for configuring the component in a functional way.
type OptionFunc func(*Component) error

// MaxMessages option for setting the max number of messages fetched.
// Allowed values are between 1 and 10.
// If messages can be processed very quickly, maxing out this value is fine, otherwise having a high value is risky as it might trigger the visibility timeout.
// Having a value too small isn't recommended either, as it increases the number of SQS API requests, thus AWS costs.
func MaxMessages(maxMessages int64) OptionFunc {
	return func(c *Component) error {
		if maxMessages <= 0 || maxMessages > 10 {
			return errors.New("max messages should be between 1 and 10")
		}
		c.cfg.maxMessages = &maxMessages
		return nil
	}
}
