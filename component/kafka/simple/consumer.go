package simple

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"golang.org/x/sync/errgroup"
)

type consumerHandler struct {
	component *Component
	pcs       map[int32]sarama.PartitionConsumer
	isBatch   bool
	ticker    *time.Ticker
	mu        sync.Mutex
	msgBuf    []*sarama.ConsumerMessage
	// Whether the handler has processed any messages
	hasProcessedMessages bool
	// WithNotificationOnceReachingLatestOffset option
	once                                sync.Once
	wg                                  sync.WaitGroup
	partitionsWithLatestOffsetUnreached map[int32]struct{}
	notificationAlreadySent             bool
}

func newConsumerHandler(component *Component, pcs map[int32]sarama.PartitionConsumer) *consumerHandler {
	partitionsWithLatestOffsetUnreached := make(map[int32]struct{}, len(pcs))
	for partition := range pcs {
		partitionsWithLatestOffsetUnreached[partition] = struct{}{}
	}

	return &consumerHandler{
		component:                           component,
		pcs:                                 pcs,
		isBatch:                             component.batchSize != 1,
		ticker:                              time.NewTicker(component.batchTimeout),
		msgBuf:                              make([]*sarama.ConsumerMessage, 0, component.batchSize),
		partitionsWithLatestOffsetUnreached: partitionsWithLatestOffsetUnreached,
	}
}

func (c *consumerHandler) consume(ctx context.Context) (hasProcessedMessages bool, err error) {
	if c.component.notificationOnceReachingLatestOffset() {
		c.wg.Add(len(c.pcs))
		go c.waitPartitionConsumersToReachLatestOffset()
	}

	eg, ctx := errgroup.WithContext(ctx)
	for partition, pc := range c.pcs {
		pc := pc
		partition := partition
		eg.Go(func() error {
			return c.consumePartition(ctx, pc, partition)
		})
	}
	return c.hasProcessedMessages, eg.Wait()
}

func (c *consumerHandler) waitPartitionConsumersToReachLatestOffset() {
	c.wg.Wait()
	// As the consumer can be retried, we have to make sure the channel is closed only once.
	c.once.Do(func() {
		close(c.component.latestOffsetReachedChan)
	})
}

func (c *consumerHandler) consumePartition(ctx context.Context, pc sarama.PartitionConsumer, partition int32) error {
	if c.component.notificationOnceReachingLatestOffset() {
		latestOffset := c.component.latestOffsets[partition]
		// We don't want to wait for consuming a message if we already know we're at the end of the stream
		if c.startingOffsetAfterLatestOffset(latestOffset, partition) {
			delete(c.partitionsWithLatestOffsetUnreached, partition)
			c.wg.Done()
		}
	}

	for {
		var err error
		if c.isBatch {
			err = c.consumePartitionBatch(ctx, pc, partition)
		} else {
			err = c.consumePartitionUnit(ctx, pc, partition)
		}
		if err != nil {
			return err
		}
	}
}

func (c *consumerHandler) startingOffsetAfterLatestOffset(latestOffset int64, partition int32) bool {
	return c.component.startingOffsets[partition] >= latestOffset
}

func (c *consumerHandler) consumePartitionUnit(ctx context.Context, pc sarama.PartitionConsumer, partition int32) error {
	select {
	case msg, ok := <-pc.Messages():
		if ok {
			log.Debugf("message claimed: value = %s, timestamp = %v, partition = %d topic = %s", string(msg.Value), msg.Timestamp, msg.Partition, msg.Topic)
			messageStatusCountInc(messageReceived, partition, msg.Topic)
			err := c.unit(ctx, msg)
			if err != nil {
				return err
			}
		} else {
			log.Debug("messages channel closed")
			return nil
		}
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			log.Infof("closing consumer: %v", ctx)
		}
		return nil
	}
	return nil
}

func (c *consumerHandler) consumePartitionBatch(ctx context.Context, pc sarama.PartitionConsumer, partition int32) error {
	select {
	case msg, ok := <-pc.Messages():
		if ok {
			log.Debugf("message claimed: value = %s, timestamp = %v, partition = %d topic = %s", string(msg.Value), msg.Timestamp, msg.Partition, msg.Topic)
			messageStatusCountInc(messageReceived, partition, msg.Topic)
			err := c.insertMessage(ctx, msg)
			if err != nil {
				return err
			}
		} else {
			log.Debug("messages channel closed")
			return nil
		}
	case <-c.ticker.C:
		c.mu.Lock()
		err := c.flush(ctx)
		c.mu.Unlock()
		if err != nil {
			return err
		}
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			log.Infof("closing consumer: %v", ctx)
		}
		return nil
	}
	return nil
}

func (c *consumerHandler) unit(ctx context.Context, msg *sarama.ConsumerMessage) error {
	messageStatusCountInc(messageProcessed, msg.Partition, msg.Topic)
	ctx, sp := c.getContextWithCorrelation(ctx, msg)
	messages := []kafka.Message{kafka.NewMessage(ctx, sp, msg)}

	btc := kafka.NewBatch(messages)
	err := c.component.proc(btc)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return fmt.Errorf("context was cancelled after processing error: %w", err)
		}
		if err = c.executeFailureStrategy(messages, err); err != nil {
			return err
		}
	}

	c.updateLatestOffsetReached(messages)
	c.hasProcessedMessages = true
	trace.SpanSuccess(messages[0].Span())

	return nil
}

func (c *consumerHandler) insertMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgBuf = append(c.msgBuf, msg)
	if len(c.msgBuf) >= int(c.component.batchSize) {
		return c.flush(ctx)
	}
	return nil
}

func (c *consumerHandler) flush(ctx context.Context) error {
	if len(c.msgBuf) == 0 {
		return nil
	}

	messages := make([]kafka.Message, len(c.msgBuf))
	for i, msg := range c.msgBuf {
		messageStatusCountInc(messageProcessed, msg.Partition, msg.Topic)
		ctx, sp := c.getContextWithCorrelation(ctx, msg)
		messages[i] = kafka.NewMessage(ctx, sp, msg)
	}

	btc := kafka.NewBatch(messages)
	err := c.component.proc(btc)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return fmt.Errorf("context was cancelled after processing error: %w", err)
		}
		if err = c.executeFailureStrategy(messages, err); err != nil {
			return err
		}
	}

	c.updateLatestOffsetReached(messages)
	c.hasProcessedMessages = true
	for _, m := range messages {
		trace.SpanSuccess(m.Span())
	}

	c.msgBuf = c.msgBuf[:0]

	return nil
}

func (c *consumerHandler) getContextWithCorrelation(ctx context.Context, msg *sarama.ConsumerMessage) (context.Context, opentracing.Span) {
	corID := getCorrelationID(msg.Headers)

	sp, ctxCh := trace.ConsumerSpan(ctx, trace.ComponentOpName(consumerComponent, msg.Topic),
		consumerComponent, corID, mapHeader(msg.Headers))
	ctxCh = correlation.ContextWithID(ctxCh, corID)
	ctxCh = log.WithContext(ctxCh, log.Sub(map[string]interface{}{correlation.ID: corID}))
	return ctxCh, sp
}

func mapHeader(rh []*sarama.RecordHeader) map[string]string {
	mp := make(map[string]string, len(rh))
	for _, h := range rh {
		mp[string(h.Key)] = string(h.Value)
	}
	return mp
}

func getCorrelationID(hh []*sarama.RecordHeader) string {
	for _, h := range hh {
		if string(h.Key) == correlation.HeaderID {
			if len(h.Value) > 0 {
				return string(h.Value)
			}
			break
		}
	}
	log.Debug("correlation header not found, creating new correlation UUID")
	return uuid.New().String()
}

func (c *consumerHandler) executeFailureStrategy(messages []kafka.Message, err error) error {
	switch c.component.failStrategy {
	case kafka.ExitStrategy:
		for _, m := range messages {
			trace.SpanError(m.Span())
			messageStatusCountInc(messageErrored, m.Message().Partition, m.Message().Topic)
		}
		log.Errorf("could not process message(s)")
		return err
	case kafka.SkipStrategy:
		for _, m := range messages {
			trace.SpanError(m.Span())
			messageStatusCountInc(messageErrored, m.Message().Partition, m.Message().Topic)
			messageStatusCountInc(messageSkipped, m.Message().Partition, m.Message().Topic)
		}
		log.Errorf("could not process message(s) so skipping with error: %v", err)
	default:
		log.Errorf("unknown failure strategy executed")
		return fmt.Errorf("unknown failure strategy: %v", c.component.failStrategy)
	}
	return nil
}

func (c *consumerHandler) updateLatestOffsetReached(messages []kafka.Message) {
	if !c.component.notificationOnceReachingLatestOffset() || c.notificationAlreadySent {
		return
	}

	// We consume from the end as it will be more efficient to find the latest offset
	for i := len(messages) - 1; i >= 0; i++ {
		message := messages[i].Message()
		partition := message.Partition
		_, latestOffsetUnreached := c.partitionsWithLatestOffsetUnreached[partition]
		if !latestOffsetUnreached {
			continue
		}

		if c.latestOffsetReached(message, partition) {
			c.wg.Done()
			delete(c.partitionsWithLatestOffsetUnreached, partition)
			if len(c.partitionsWithLatestOffsetUnreached) == 0 {
				c.notificationAlreadySent = true
				return
			}
		}
	}
}

func (c *consumerHandler) latestOffsetReached(message *sarama.ConsumerMessage, partition int32) bool {
	return message.Offset >= c.component.latestOffsets[partition]
}
