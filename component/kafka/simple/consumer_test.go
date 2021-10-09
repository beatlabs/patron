package simple

import (
	"context"
	"testing"
	"time"

	"github.com/beatlabs/patron/component/kafka"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumePartition(t *testing.T) {
	messages := []*sarama.ConsumerMessage{
		msg(0, 0),
		msg(0, 1),
		msg(1, 0),
		msg(1, 1),
		msg(1, 2),
		msg(2, 1),
		msg(2, 2),
	}
	expectedMessages := map[int32]map[int64]struct{}{
		0: {0: {}, 1: {}},
		1: {0: {}, 1: {}, 2: {}},
		2: {1: {}, 2: {}},
	}

	type args struct {
		ctx       context.Context
		ch        <-chan *sarama.ConsumerMessage
		partition int32
		component *Component
	}
	tests := map[string]struct {
		args             args
		expectedMessages map[int32]map[int64]struct{}
		expectedErr      error
	}{
		"success - unit": {
			args: args{
				ctx: context.Background(),
				component: &Component{
					batchSize:    1,
					batchTimeout: defaultBatchTimeout,
				},
				ch: toChannelEventuallyClosing(messages...),
			},
			expectedMessages: expectedMessages,
		},
		"success - batch size smaller than number of messages": {
			args: args{
				ctx: context.Background(),
				component: &Component{
					batchSize:    3,
					batchTimeout: defaultBatchTimeout,
				},
				ch: toChannelEventuallyClosing(messages...),
			},
			expectedMessages: expectedMessages,
		},
		"success - batch size greater than number of messages - channel closing": {
			args: args{
				ctx: context.Background(),
				component: &Component{
					batchSize:    100,
					batchTimeout: defaultBatchTimeout,
				},
				ch: toChannelEventuallyClosing(messages...),
			},
			expectedMessages: expectedMessages,
		},
		"success - batch size greater than number of messages - channel not closing": {
			args: args{
				ctx: ctxTimeout(time.Second),
				component: &Component{
					batchSize:    100,
					batchTimeout: defaultBatchTimeout,
				},
				ch: toChannelNotClosing(messages...),
			},
			expectedMessages: expectedMessages,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mock := consumerMock{}
			tt.args.component.proc = mock.record
			handler := newConsumerHandler(tt.args.component, nil)
			err := handler.consumePartition(tt.args.ctx, tt.args.ch, tt.args.partition)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
				mock.assert(t, tt.expectedMessages)
			}
		})
	}
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

type consumerMock struct {
	// partition / offset
	got map[int32]map[int64]struct{}
}

func (m *consumerMock) record(batch kafka.Batch) error {
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
