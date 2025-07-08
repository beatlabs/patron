// Package kafka provides a client with included tracing capabilities.
package v2

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	deliveryTypeAsync = "async"
)

var deliveryTypeAsyncAttr = attribute.String("delivery", deliveryTypeAsync)

// AsyncProducer is an asynchronous Kafka producer.
type AsyncProducer struct {
	producer sarama.AsyncProducer
}

// AsyncClose triggers a shutdown of the producer. The shutdown has completed
// when both the Errors and Successes channels have been closed. When calling
// AsyncClose, you *must* continue to read from those channels in order to
// drain the results of any messages in flight.
func (ap *AsyncProducer) AsyncClose() {
	ap.producer.AsyncClose()
}

// Close shuts down the producer and waits for any buffered messages to be
// flushed. You must call this function before a producer object passes out of
// scope, as it may otherwise leak memory. You must call this before process
// shutting down, or you may lose messages. You must call this before calling
// Close on the underlying client.
func (ap *AsyncProducer) Close() error {
	return ap.producer.Close()
}

// Send a message to a topic, asynchronously. Producer errors are queued on the
// channel obtained during the AsyncProducer creation.
func (ap *AsyncProducer) Send(ctx context.Context, msg *sarama.ProducerMessage) error {
	ctx, sp := startSpan(ctx, "send", deliveryTypeAsync, msg.Topic)
	defer sp.End()

	injectTracingAndCorrelationHeaders(ctx, msg)

	ap.producer.Input() <- msg
	publishCountAdd(ctx, deliveryTypeAsyncAttr, observability.SucceededAttribute, topicAttribute(msg.Topic))
	sp.SetStatus(codes.Ok, "message sent")
	return nil
}

// Successes is the success output channel back to the user when Return.Successes is
// enabled. If Return.Successes is true, you MUST read from this channel or the
// Producer will deadlock. It is suggested that you send and read messages
// together in a single select statement.
func (ap *AsyncProducer) Successes() <-chan *sarama.ProducerMessage {
	return ap.producer.Successes()
}

// Errors is the error output channel back to the user. You MUST read from this
// channel or the Producer will deadlock when the channel is full. Alternatively,
// you can set Producer.Return.Errors in your config to false, which prevents
// errors to be returned.
func (ap *AsyncProducer) Errors() <-chan *sarama.ProducerError {
	return ap.producer.Errors()
}

// IsTransactional return true when current producer is transactional.
func (ap *AsyncProducer) IsTransactional() bool {
	return ap.producer.IsTransactional()
}

// TxnStatus return current producer transaction status.
func (ap *AsyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	return ap.producer.TxnStatus()
}

// BeginTxn mark current transaction as ready.
func (ap *AsyncProducer) BeginTxn() error {
	return ap.producer.BeginTxn()
}

// CommitTxn commit current transaction.
func (ap *AsyncProducer) CommitTxn() error {
	return ap.producer.CommitTxn()
}

// AbortTxn abort current transaction.
func (ap *AsyncProducer) AbortTxn() error {
	return ap.producer.AbortTxn()
}

// AddOffsetsToTxn add associated offsets to current transaction.
func (ap *AsyncProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupId string) error {
	return ap.producer.AddOffsetsToTxn(offsets, groupId)
}

// AddMessageToTxn add message offsets to current transaction.
func (ap *AsyncProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupId string, metadata *string) error {
	return ap.producer.AddMessageToTxn(msg, groupId, metadata)
}
