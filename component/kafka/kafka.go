package kafka

import (
	"context"

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
