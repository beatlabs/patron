// Package kafka provides a client with included tracing capabilities.
package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel/codes"
)

// AsyncProducer is an asynchronous Kafka producer.
type AsyncProducer struct {
	baseProducer
	asyncProd sarama.AsyncProducer
}

// Send a message to a topic, asynchronously. Producer errors are queued on the
// channel obtained during the AsyncProducer creation.
func (ap *AsyncProducer) Send(ctx context.Context, msg *sarama.ProducerMessage) error {
	ctx, sp := startSpan(ctx, "send", deliveryTypeAsync, msg.Topic)
	defer sp.End()

	injectTracingAndCorrelationHeaders(ctx, msg)

	ap.asyncProd.Input() <- msg
	statusCountAdd(deliveryTypeAsync, deliveryStatusSent, msg.Topic)
	sp.SetStatus(codes.Ok, "message sent")
	return nil
}

func (ap *AsyncProducer) propagateError(chErr chan<- error) {
	for pe := range ap.asyncProd.Errors() {
		statusCountAdd(deliveryTypeAsync, deliveryStatusSendError, pe.Msg.Topic)
		chErr <- fmt.Errorf("failed to send message: %w", pe)
	}
}

// Close shuts down the producer and waits for any buffered messages to be
// flushed. You must call this function before a producer object passes out of
// scope, as it may otherwise leak memory.
func (ap *AsyncProducer) Close() error {
	if err := ap.asyncProd.Close(); err != nil {
		return errors.Join(fmt.Errorf("failed to close async producer client: %w", err), ap.prodClient.Close())
	}
	if err := ap.prodClient.Close(); err != nil {
		return fmt.Errorf("failed to close async producer: %w", err)
	}
	return nil
}
