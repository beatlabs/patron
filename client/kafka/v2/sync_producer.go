package v2

import (
	"context"
	"errors"
	"fmt"

	"github.com/Shopify/sarama"
	patronerrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const batchTarget = "batch"

var syncTag = opentracing.Tag{Key: "type", Value: deliveryTypeSync}

// SyncProducer is a synchronous Kafka producer.
type SyncProducer struct {
	baseProducer
	syncProd sarama.SyncProducer
}

// Send a message to a topic.
func (p *SyncProducer) Send(ctx context.Context, msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	sp, _ := trace.ChildSpan(ctx, trace.ComponentOpName(componentTypeSync, msg.Topic), componentTypeSync,
		ext.SpanKindProducer, syncTag, opentracing.Tag{Key: "topic", Value: msg.Topic})

	err = injectTracingHeaders(msg, sp)
	if err != nil {
		statusCountAdd(deliveryTypeSync, deliveryStatusSendError, msg.Topic, 1)
		trace.SpanError(sp)
		return -1, -1, fmt.Errorf("failed to inject tracing headers: %w", err)
	}

	partition, offset, err = p.syncProd.SendMessage(msg)
	if err != nil {
		statusCountAdd(deliveryTypeSync, deliveryStatusSendError, msg.Topic, 1)
		trace.SpanError(sp)
		return -1, -1, err
	}

	statusCountAdd(deliveryTypeSync, deliveryStatusSent, msg.Topic, 1)
	trace.SpanSuccess(sp)
	return partition, offset, nil
}

// SendBatch sends a batch to a topic.
func (p *SyncProducer) SendBatch(ctx context.Context, messages []*sarama.ProducerMessage) error {
	if len(messages) == 0 {
		return errors.New("messages are empty or nil")
	}

	sp, _ := trace.ChildSpan(ctx, trace.ComponentOpName(componentTypeSync, batchTarget), componentTypeSync,
		ext.SpanKindProducer, syncTag, opentracing.Tag{Key: "topic", Value: batchTarget})

	for _, msg := range messages {
		if err := injectTracingHeaders(msg, sp); err != nil {
			statusCountAdd(deliveryTypeSync, deliveryStatusSendError, batchTarget, len(messages))
			trace.SpanError(sp)
			return fmt.Errorf("failed to inject tracing headers: %w", err)
		}
	}

	if err := p.syncProd.SendMessages(messages); err != nil {
		statusCountAdd(deliveryTypeSync, deliveryStatusSendError, batchTarget, len(messages))
		trace.SpanError(sp)
		return err
	}

	statusCountAdd(deliveryTypeSync, deliveryStatusSent, batchTarget, len(messages))
	trace.SpanSuccess(sp)
	return nil
}

// Close shuts down the producer and waits for any buffered messages to be
// flushed. You must call this function before a producer object passes out of
// scope, as it may otherwise leak memory.
func (p *SyncProducer) Close() error {
	if err := p.syncProd.Close(); err != nil {
		return patronerrors.Aggregate(fmt.Errorf("failed to close sync producer client: %w", err), p.prodClient.Close())
	}
	if err := p.prodClient.Close(); err != nil {
		return fmt.Errorf("failed to close sync producer: %w", err)
	}
	return nil
}
