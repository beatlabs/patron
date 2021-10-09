package simple

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumePartition(t *testing.T) {
	messages := map[int32][]*sarama.ConsumerMessage{
		0: {
			msg(0, 0),
			msg(0, 1),
		},
		1: {
			msg(1, 0),
			msg(1, 1),
			msg(1, 2),
		},
		2: {
			msg(2, 1),
			msg(2, 2),
		},
	}
	expectedMessages := map[int32]map[int64]struct{}{
		0: {0: {}, 1: {}},
		1: {0: {}, 1: {}, 2: {}},
		2: {1: {}, 2: {}},
	}
	nbMessages := 7
	startingOffsets := map[int32]int64{0: 0, 1: 0, 2: 1}
	latestOffsets := map[int32]int64{0: 1, 1: 2, 2: 2}

	type args struct {
		ctx             context.Context
		procMock        *consumerMock
		pcs             map[int32]sarama.PartitionConsumer
		partition       int32
		component       *Component
		notifCh         chan struct{}
		startingOffsets map[int32]int64
		latestOffsets   map[int32]int64
	}
	tests := map[string]struct {
		args                      args
		expectedMessages          map[int32]map[int64]struct{}
		expectedProcessedMessages bool
		expectNotifChannelClosed  bool
		expectedErr               error
	}{
		"success - unit": {
			args: args{
				ctx:      context.Background(),
				procMock: &consumerMock{},
				component: &Component{
					batchSize:    1,
					batchTimeout: defaultBatchTimeout,
				},
				pcs:             createPartitionConsumersWithClosure(messages),
				notifCh:         make(chan struct{}),
				startingOffsets: startingOffsets,
				latestOffsets:   latestOffsets,
			},
			expectedProcessedMessages: true,
			expectNotifChannelClosed:  true,
			expectedMessages:          expectedMessages,
		},
		"success - context cancelled - no messages": {
			args: args{
				ctx:      ctxAlreadyClosed(),
				procMock: &consumerMock{},
				component: &Component{
					batchSize:    1,
					batchTimeout: defaultBatchTimeout,
				},
				pcs:             createPartitionConsumersWithoutClosuse(nil),
				notifCh:         make(chan struct{}),
				startingOffsets: startingOffsets,
				latestOffsets:   latestOffsets,
			},
			expectedProcessedMessages: false,
			expectNotifChannelClosed:  false,
		},
		"success - batch size smaller than number of messages": {
			args: args{
				ctx:      context.Background(),
				procMock: &consumerMock{},
				component: &Component{
					batchSize:    3,
					batchTimeout: defaultBatchTimeout,
				},
				pcs:             createPartitionConsumersWithClosure(messages),
				notifCh:         make(chan struct{}),
				startingOffsets: startingOffsets,
				latestOffsets:   latestOffsets,
			},
			expectedProcessedMessages: true,
			expectNotifChannelClosed:  true,
			expectedMessages:          expectedMessages,
		},
		"success - batch size greater than number of messages": {
			args: args{
				ctx:      context.Background(),
				procMock: &consumerMock{},
				component: &Component{
					batchSize:    100,
					batchTimeout: defaultBatchTimeout,
				},
				pcs:             createPartitionConsumersWithClosure(messages),
				notifCh:         make(chan struct{}),
				startingOffsets: startingOffsets,
				latestOffsets:   latestOffsets,
			},
			expectedProcessedMessages: true,
			expectNotifChannelClosed:  true,
			expectedMessages:          expectedMessages,
		},
		"success - batch size greater than number of messages - no closure": {
			args: args{
				ctx:      ctxTimeout(time.Second),
				procMock: &consumerMock{},
				component: &Component{
					batchSize:    100,
					batchTimeout: defaultBatchTimeout,
				},
				pcs:             createPartitionConsumersWithoutClosuse(messages),
				notifCh:         make(chan struct{}),
				startingOffsets: startingOffsets,
				latestOffsets:   latestOffsets,
			},
			expectedProcessedMessages: true,
			expectNotifChannelClosed:  true,
			expectedMessages:          expectedMessages,
		},
		"success - starting offsets already reached": {
			args: args{
				ctx:      context.Background(),
				procMock: &consumerMock{},
				component: &Component{
					batchSize:    1,
					batchTimeout: defaultBatchTimeout,
				},
				pcs:             createPartitionConsumersWithClosure(messages),
				notifCh:         make(chan struct{}),
				startingOffsets: map[int32]int64{0: 1, 1: 2, 2: 2},
				latestOffsets:   latestOffsets,
			},
			expectedProcessedMessages: true,
			expectNotifChannelClosed:  true,
			expectedMessages:          expectedMessages,
		},
		"success - unit - failure strategy - skip": {
			args: args{
				ctx:      context.Background(),
				procMock: &consumerMock{errors: nbMessages},
				component: &Component{
					batchSize:    1,
					batchTimeout: defaultBatchTimeout,
					failStrategy: kafka.SkipStrategy,
				},
				pcs:             createPartitionConsumersWithClosure(messages),
				notifCh:         make(chan struct{}),
				startingOffsets: startingOffsets,
				latestOffsets:   latestOffsets,
			},
			expectedProcessedMessages: true,
			expectNotifChannelClosed:  true,
		},
		"success - batch - failure strategy - skip": {
			args: args{
				ctx:      context.Background(),
				procMock: &consumerMock{errors: nbMessages},
				component: &Component{
					batchSize:    3,
					batchTimeout: defaultBatchTimeout,
					failStrategy: kafka.SkipStrategy,
				},
				pcs:             createPartitionConsumersWithClosure(messages),
				notifCh:         make(chan struct{}),
				startingOffsets: startingOffsets,
				latestOffsets:   latestOffsets,
			},
			expectedProcessedMessages: true,
			expectNotifChannelClosed:  true,
		},
		"error - unit": {
			args: args{
				ctx:      context.Background(),
				procMock: &consumerMock{errors: 1},
				component: &Component{
					batchSize:    1,
					batchTimeout: defaultBatchTimeout,
					failStrategy: kafka.ExitStrategy,
				},
				pcs: createPartitionConsumersWithClosure(messages),
			},
			expectedProcessedMessages: false,
			expectNotifChannelClosed:  true,
			expectedErr:               errors.New("proc error"),
		},
		"error - batch": {
			args: args{
				ctx:      context.Background(),
				procMock: &consumerMock{errors: 1},
				component: &Component{
					batchSize:    3,
					batchTimeout: defaultBatchTimeout,
				},
				pcs: createPartitionConsumersWithClosure(messages),
			},
			expectedProcessedMessages: false,
			expectNotifChannelClosed:  true,
			expectedErr:               errors.New("proc error"),
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tt.args.component.proc = tt.args.procMock.record
			if tt.args.notifCh != nil {
				tt.args.component.latestOffsetReachedChan = tt.args.notifCh
				tt.args.component.startingOffsets = tt.args.startingOffsets
				tt.args.component.latestOffsets = tt.args.latestOffsets
			}

			handler := newConsumerHandler(tt.args.component, tt.args.pcs)
			hasProcessedMessages, err := handler.consume(tt.args.ctx)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectNotifChannelClosed, isChannelClosed(tt.args.notifCh))
				tt.args.procMock.assert(t, tt.expectedMessages)
				assert.Equal(t, tt.expectedProcessedMessages, hasProcessedMessages)
			}
		})
	}
}

func createPartitionConsumersWithClosure(in map[int32][]*sarama.ConsumerMessage) map[int32]sarama.PartitionConsumer {
	res := make(map[int32]sarama.PartitionConsumer, len(in))
	for partition, consumerMessages := range in {
		res[partition] = partitionConsumerStub{ch: toChannelEventuallyClosing(consumerMessages...)}
	}
	return res
}

func createPartitionConsumersWithoutClosuse(in map[int32][]*sarama.ConsumerMessage) map[int32]sarama.PartitionConsumer {
	res := make(map[int32]sarama.PartitionConsumer, len(in))
	for partition, consumerMessages := range in {
		res[partition] = partitionConsumerStub{ch: toChannelNotClosing(consumerMessages...)}
	}
	return res
}

type partitionConsumerStub struct {
	ch <-chan *sarama.ConsumerMessage
}

func (p partitionConsumerStub) AsyncClose() {}

func (p partitionConsumerStub) Close() error {
	return nil
}

func (p partitionConsumerStub) Messages() <-chan *sarama.ConsumerMessage {
	return p.ch
}

func (p partitionConsumerStub) Errors() <-chan *sarama.ConsumerError {
	return nil
}

func (p partitionConsumerStub) HighWaterMarkOffset() int64 {
	return 0
}

func msg(partition int32, offset int64) *sarama.ConsumerMessage {
	return &sarama.ConsumerMessage{
		Partition: partition,
		Offset:    offset,
	}
}

func toChannelEventuallyClosing(msgs ...*sarama.ConsumerMessage) <-chan *sarama.ConsumerMessage {
	ch := make(chan *sarama.ConsumerMessage, len(msgs))
	go func() {
		for _, msg := range msgs {
			ch <- msg
		}
		close(ch)
	}()
	return ch
}

func toChannelNotClosing(msgs ...*sarama.ConsumerMessage) <-chan *sarama.ConsumerMessage {
	ch := make(chan *sarama.ConsumerMessage, len(msgs))
	go func() {
		for _, msg := range msgs {
			ch <- msg
		}
	}()
	return ch
}

func ctxTimeout(d time.Duration) context.Context {
	ctx, _ := context.WithTimeout(context.Background(), d)
	return ctx
}

func ctxAlreadyClosed() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

type consumerMock struct {
	mu sync.Mutex
	// partition / offset
	got    map[int32]map[int64]struct{}
	errors int
}

func (m *consumerMock) record(batch kafka.Batch) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errors > 0 {
		m.errors--
		return errors.New("proc error")
	}

	if m.got == nil {
		m.got = make(map[int32]map[int64]struct{})
	}

	msgs := batch.Messages()
	for _, msg := range msgs {
		offsets, exists := m.got[msg.Message().Partition]
		if !exists {
			offsets = make(map[int64]struct{})
			m.got[msg.Message().Partition] = offsets
		}

		if _, exists = offsets[msg.Message().Offset]; !exists {
			offsets[msg.Message().Offset] = struct{}{}
		}
	}
	return nil
}

func (m *consumerMock) assert(t *testing.T, expected map[int32]map[int64]struct{}) {
	assert.EqualValues(t, expected, m.got)
}

func isChannelClosed(ch <-chan struct{}) bool {
	select {
	case _, ok := <-ch:
		if !ok {
			return true
		}
	default:
	}
	return false
}
