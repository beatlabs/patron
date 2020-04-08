package kafka

import (
	"context"
	"fmt"

	"github.com/beatlabs/patron/trace"

	"github.com/Shopify/sarama"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// SyncProducer is a synchronous Kafka producer.
type SyncProducer struct {
	baseProducer

	syncProd sarama.SyncProducer
}

// Send a message to a topic.
func (p *SyncProducer) Send(ctx context.Context, msg *Message) error {
	sp, _ := trace.ChildSpan(ctx, trace.ComponentOpName(syncProducerComponent, msg.topic),
		syncProducerComponent, ext.SpanKindProducer, p.tag,
		opentracing.Tag{Key: "topic", Value: msg.topic})
	pm, err := p.createProducerMessage(ctx, msg, sp)
	if err != nil {
		p.statusCountInc(messageCreationErrors, msg.topic)
		trace.SpanError(sp)
		return err
	}

	_, _, err = p.syncProd.SendMessage(pm)
	if err != nil {
		p.statusCountInc(messageCreationErrors, msg.topic)
		trace.SpanError(sp)
		return err
	}

	p.statusCountInc(messageSent, msg.topic)
	trace.SpanSuccess(sp)

	return nil
}

// Close shuts down the producer and waits for any buffered messages to be
// flushed. You must call this function before a producer object passes out of
// scope, as it may otherwise leak memory.
func (p *SyncProducer) Close() error {
	err := p.syncProd.Close()
	if err != nil {
		// always close client
		_ = p.prodClient.Close()

		return fmt.Errorf("failed to close sync producer client: %w", err)
	}

	err = p.prodClient.Close()
	if err != nil {
		return fmt.Errorf("failed to close sync producer: %w", err)
	}
	return nil
}
