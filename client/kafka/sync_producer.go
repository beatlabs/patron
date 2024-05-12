package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel/codes"
)

// SyncProducer is a synchronous Kafka producer.
type SyncProducer struct {
	baseProducer
	syncProd sarama.SyncProducer
}

// Send a message to a topic.
func (p *SyncProducer) Send(ctx context.Context, msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	ctx, sp := startSpan(ctx, "send", deliveryTypeSync, msg.Topic)
	defer sp.End()

	injectTracingAndCorrelationHeaders(ctx, msg)

	partition, offset, err = p.syncProd.SendMessage(msg)
	if err != nil {
		statusCountAdd(deliveryTypeSync, deliveryStatusSendError, msg.Topic, 1)
		sp.RecordError(err)
		sp.SetStatus(codes.Error, "error sending message")
		return -1, -1, err
	}

	statusCountAdd(deliveryTypeSync, deliveryStatusSent, msg.Topic, 1)
	sp.SetStatus(codes.Ok, "message sent")
	return partition, offset, nil
}

// SendBatch sends a batch to a topic.
func (p *SyncProducer) SendBatch(ctx context.Context, messages []*sarama.ProducerMessage) error {
	if len(messages) == 0 {
		return errors.New("messages are empty or nil")
	}

	ctx, sp := startSpan(ctx, "send-batch", deliveryTypeSync, "")
	defer sp.End()

	for _, msg := range messages {
		injectTracingAndCorrelationHeaders(ctx, msg)
	}

	if err := p.syncProd.SendMessages(messages); err != nil {
		statusCountBatchAdd(deliveryTypeSync, deliveryStatusSendError, messages)
		sp.RecordError(err)
		sp.SetStatus(codes.Error, "error sending batch")
		return err
	}

	statusCountBatchAdd(deliveryTypeSync, deliveryStatusSent, messages)
	sp.SetStatus(codes.Ok, "batch sent")
	return nil
}

// Close shuts down the producer and waits for any buffered messages to be
// flushed. You must call this function before a producer object passes out of
// scope, as it may otherwise leak memory.
func (p *SyncProducer) Close() error {
	if err := p.syncProd.Close(); err != nil {
		return errors.Join(fmt.Errorf("failed to close sync producer client: %w", err), p.prodClient.Close())
	}
	if err := p.prodClient.Close(); err != nil {
		return fmt.Errorf("failed to close sync producer: %w", err)
	}
	return nil
}

func statusCountBatchAdd(deliveryType string, status deliveryStatus, messages []*sarama.ProducerMessage) {
	for _, msg := range messages {
		statusCountAdd(deliveryType, status, msg.Topic, 1)
	}
}
