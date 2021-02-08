package kafka

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/correlation"
	"github.com/google/uuid"
)

// FailStrategy type definition.
type FailStrategy int

const (
	// ExitStrategy does not commit failed message offsets and exits the application.
	ExitStrategy FailStrategy = iota
	// SkipStrategy commits the offset of messages that failed processing, and continues processing.
	SkipStrategy
)

// BatchProcessorFunc definition of a batch async processor.
type BatchProcessorFunc func(context.Context, []MessageWrapper) error

// MessageWrapper interface for wrapping messages that are handled by the kafka component.
type MessageWrapper interface {
	GetConsumerMessage() *sarama.ConsumerMessage
	GetCorrelationID() string
}

type messageWrapper struct {
	msg *sarama.ConsumerMessage
}

// GetConsumerMessage gets the raw consumer message received via the kafka component.
func (m *messageWrapper) GetConsumerMessage() *sarama.ConsumerMessage {
	return m.msg
}

// GetCorrelationID tries to fetch the correlation ID from the message headers and if the header was missing
// it will generate a new correlation ID.
func (m *messageWrapper) GetCorrelationID() string {
	for _, h := range m.msg.Headers {
		if string(h.Key) == correlation.HeaderID {
			if len(h.Value) > 0 {
				return string(h.Value)
			}
			break
		}
	}
	return uuid.New().String()
}

// defaultSaramaConfig function creates a sarama config object with the default configuration set up.
func defaultSaramaConfig(name string) (*sarama.Config, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, errors.New("failed to get hostname")
	}

	config := sarama.NewConfig()
	config.ClientID = fmt.Sprintf("%s-%s", host, name)
	config.Consumer.Return.Errors = true
	config.Version = sarama.V0_11_0_0

	return config, nil
}
