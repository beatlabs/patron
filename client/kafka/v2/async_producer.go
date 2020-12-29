// Package v2 provides a client with included tracing capabilities.
package v2

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	patronerrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/trace"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

var asyncTag = opentracing.Tag{Key: "type", Value: deliveryTypeAsync}

// AsyncProducer is an asynchronous Kafka producer.
type AsyncProducer struct {
	baseProducer
	asyncProd sarama.AsyncProducer
}

// Send a message to a topic, asynchronously. Producer errors are queued on the
// channel obtained during the AsyncProducer creation.
func (ap *AsyncProducer) Send(ctx context.Context, msg *sarama.ProducerMessage) error {
	sp, _ := trace.ChildSpan(ctx, trace.ComponentOpName(componentTypeAsync, msg.Topic), componentTypeAsync,
		ext.SpanKindProducer, asyncTag, opentracing.Tag{Key: "topic", Value: msg.Topic})

	err := injectTracingHeaders(msg, sp)
	if err != nil {
		statusCountInc(deliveryTypeAsync, deliveryStatusCreationError, msg.Topic)
		trace.SpanError(sp)
		return fmt.Errorf("failed to inject tracing headers: %w", err)
	}

	ap.asyncProd.Input() <- msg
	statusCountInc(deliveryTypeAsync, deliveryStatusSent, msg.Topic)
	trace.SpanSuccess(sp)
	return nil
}

func injectTracingHeaders(msg *sarama.ProducerMessage, sp opentracing.Span) error {
	c := kafkaHeadersCarrier(msg.Headers)

	return sp.Tracer().Inject(sp.Context(), opentracing.TextMap, &c)
}

func (ap *AsyncProducer) propagateError(chErr chan<- error) {
	for pe := range ap.asyncProd.Errors() {
		statusCountInc(deliveryTypeAsync, deliveryStatusSendError, pe.Msg.Topic)
		chErr <- fmt.Errorf("failed to send message: %w", pe)
	}
}

// Close shuts down the producer and waits for any buffered messages to be
// flushed. You must call this function before a producer object passes out of
// scope, as it may otherwise leak memory.
func (ap *AsyncProducer) Close() error {
	if err := ap.asyncProd.Close(); err != nil {
		return patronerrors.Aggregate(fmt.Errorf("failed to close async producer client: %w", err), ap.prodClient.Close())
	}
	if err := ap.prodClient.Close(); err != nil {
		return fmt.Errorf("failed to close async producer: %w", err)
	}
	return nil
}
