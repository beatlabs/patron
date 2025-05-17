package v2

import (
	"context"
	"errors"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	deliveryTypeSync = "sync"
)

var deliveryTypeSyncAttr = attribute.String("delivery", deliveryTypeSync)

// SyncProducer is a synchronous Kafka producer.
type SyncProducer struct {
	producer sarama.SyncProducer
}

// SendMessage produces a given message, and returns only when it either has
// succeeded or failed to produce. It will return the partition and the offset
// of the produced message, or an error if the message failed to produce.
func (p *SyncProducer) SendMessage(ctx context.Context, msg *sarama.ProducerMessage) (int32, int64, error) {
	ctx, sp := startSpan(ctx, "send", deliveryTypeSync, msg.Topic)
	defer sp.End()

	injectTracingAndCorrelationHeaders(ctx, msg)

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		publishCountAdd(ctx, deliveryTypeSyncAttr, observability.FailedAttribute, topicAttribute(msg.Topic))
		sp.RecordError(err)
		sp.SetStatus(codes.Error, "error sending message")
		return -1, -1, err
	}

	publishCountAdd(ctx, deliveryTypeSyncAttr, observability.SucceededAttribute, topicAttribute(msg.Topic))
	sp.SetStatus(codes.Ok, "message sent")
	return partition, offset, nil
}

// SendMessages produces a given set of messages, and returns only when all
// messages in the set have either succeeded or failed. Note that messages
// can succeed and fail individually; if some succeed and some fail,
// SendMessages will return an error.
func (p *SyncProducer) SendBatch(ctx context.Context, messages []*sarama.ProducerMessage) error {
	if len(messages) == 0 {
		return errors.New("messages are empty or nil")
	}

	ctx, sp := startSpan(ctx, "send-batch", deliveryTypeSync, "")
	defer sp.End()

	for _, msg := range messages {
		injectTracingAndCorrelationHeaders(ctx, msg)
	}

	if err := p.producer.SendMessages(messages); err != nil {
		statusCountBatchAdd(ctx, observability.FailedAttribute, messages)
		sp.RecordError(err)
		sp.SetStatus(codes.Error, "error sending batch")
		return err
	}

	statusCountBatchAdd(ctx, observability.SucceededAttribute, messages)
	sp.SetStatus(codes.Ok, "batch sent")
	return nil
}

// Close shuts down the producer and waits for any buffered messages to be
// flushed. You must call this function before a producer object passes out of
// scope, as it may otherwise leak memory.
func (p *SyncProducer) Close() error {
	return p.producer.Close()
}

// TxnStatus return current producer transaction status.
func (p *SyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	return p.producer.TxnStatus()
}

// IsTransactional return true when current producer is transactional.
func (p *SyncProducer) IsTransactional() bool {
	return p.producer.IsTransactional()
}

// BeginTxn mark current transaction as ready.
func (p *SyncProducer) BeginTxn() error {
	return p.producer.BeginTxn()
}

// CommitTxn commit current transaction.
func (p *SyncProducer) CommitTxn() error {
	return p.producer.CommitTxn()
}

// AbortTxn abort current transaction.
func (p *SyncProducer) AbortTxn() error {
	return p.producer.AbortTxn()
}

// AddOffsetsToTxn add associated offsets to current transaction.
func (p *SyncProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupId string) error {
	return p.producer.AddOffsetsToTxn(offsets, groupId)
}

// AddMessageToTxn add message offsets to current transaction.
func (p *SyncProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupId string, metadata *string) error {
	return p.producer.AddMessageToTxn(msg, groupId, metadata)
}
