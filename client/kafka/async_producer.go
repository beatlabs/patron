package kafka

import (
	"context"
	"fmt"

	"github.com/beatlabs/patron/trace"

	"github.com/Shopify/sarama"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// AsyncProducer is an asynchronous Kafka producer.
type AsyncProducer struct {
	baseProducer
	asyncProd sarama.AsyncProducer
	chErr     chan error
}

// Send a message to a topic, asynchronously. Producer errors are queued on the
// channel obtained during the AsyncProducer creation.
func (ap *AsyncProducer) Send(ctx context.Context, msg *Message) error {
	sp, _ := trace.ChildSpan(ctx, trace.ComponentOpName(asyncProducerComponent, msg.topic),
		asyncProducerComponent, ext.SpanKindProducer, ap.tag,
		opentracing.Tag{Key: "topic", Value: msg.topic})
	pm, err := ap.createProducerMessage(ctx, msg, sp)
	if err != nil {
		ap.statusCountInc(messageCreationErrors, msg.topic)
		trace.SpanError(sp)
		return err
	}

	ap.statusCountInc(messageSent, msg.topic)
	ap.asyncProd.Input() <- pm
	trace.SpanSuccess(sp)

	return nil
}

func (ap *AsyncProducer) propagateError() {
	for pe := range ap.asyncProd.Errors() {
		ap.statusCountInc(messageSendErrors, pe.Msg.Topic)
		ap.chErr <- fmt.Errorf("failed to send message: %w", pe)
	}
}

// Close shuts down the producer and waits for any buffered messages to be
// flushed. You must call this function before a producer object passes out of
// scope, as it may otherwise leak memory.
func (ap *AsyncProducer) Close() error {
	err := ap.asyncProd.Close()
	if err != nil {
		// always close client
		_ = ap.prodClient.Close()

		return fmt.Errorf("failed to close async producer client: %w", err)
	}

	err = ap.prodClient.Close()
	if err != nil {
		return fmt.Errorf("failed to close async producer: %w", err)
	}
	return nil
}
