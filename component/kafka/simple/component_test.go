package simple

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	type args struct {
		name    string
		brokers []string
		topic   string
		proc    kafka.BatchProcessorFunc
		options OptionFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr bool
	}{
		"success": {
			args: args{
				name:    "name",
				brokers: []string{"localhost"},
				topic:   "topic",
				proc:    func(batch kafka.Batch) error { return nil },
				options: FailureStrategy(kafka.ExitStrategy),
			},
		},
		"missing name": {
			args: args{
				brokers: []string{"localhost"},
				topic:   "topic",
				proc:    func(batch kafka.Batch) error { return nil },
			},
			expectedErr: true,
		},
		"missing broker": {
			args: args{
				name:  "name",
				topic: "topic",
				proc:  func(batch kafka.Batch) error { return nil },
			},
			expectedErr: true,
		},
		"missing topic": {
			args: args{
				name:    "name",
				brokers: []string{"localhost"},
				proc:    func(batch kafka.Batch) error { return nil },
			},
			expectedErr: true,
		},
		"missing proc": {
			args: args{
				name:    "name",
				brokers: []string{"localhost"},
				topic:   "topic",
			},
			expectedErr: true,
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg, err := kafka.DefaultConsumerSaramaConfig("test", true)
			require.NoError(t, err)
			comp, err := New(tt.args.name, tt.args.brokers, tt.args.topic, tt.args.proc, cfg)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, kafka.ExitStrategy, comp.failStrategy)
			}
		})
	}
}

func TestRun(t *testing.T) {
	singleMessage := map[int32][]*sarama.ConsumerMessage{
		0: {
			msg(0, 0),
		},
	}
	expectedSingleMessage := map[int32]map[int64]struct{}{
		0: {0: {}},
	}
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

	type args struct {
		ctx                      context.Context
		procMock                 *consumerMock
		partitionConsumerFactory func(ctx context.Context) (sarama.Client, sarama.Consumer, map[int32]sarama.PartitionConsumer, error)
		options                  []OptionFunc
	}
	tests := map[string]struct {
		args             args
		expectedMessages map[int32]map[int64]struct{}
		expectedErr      error
	}{
		"success - no retry": {
			args: args{
				ctx:      context.Background(),
				procMock: &consumerMock{},
				partitionConsumerFactory: func(ctx context.Context) (sarama.Client, sarama.Consumer, map[int32]sarama.PartitionConsumer, error) {
					return clientStub{}, consumerStub{}, createPartitionConsumersWithClosure(messages), nil
				},
				options: []OptionFunc{Retries(1)},
			},
			expectedMessages: expectedMessages,
		},
		"error - no retry": {
			args: args{
				ctx:      context.Background(),
				procMock: &consumerMock{errors: 1},
				partitionConsumerFactory: func(ctx context.Context) (sarama.Client, sarama.Consumer, map[int32]sarama.PartitionConsumer, error) {
					return clientStub{}, consumerStub{}, createPartitionConsumersWithClosure(messages), nil
				},
				options: []OptionFunc{Retries(1)},
			},
			expectedErr: errors.New("simple consumer error on topic topic: proc error"),
		},
		"success - retry": {
			args: args{
				ctx:      ctxTimeout(time.Second),
				procMock: &consumerMock{errors: 1},
				partitionConsumerFactory: func(ctx context.Context) (sarama.Client, sarama.Consumer, map[int32]sarama.PartitionConsumer, error) {
					return clientStub{}, consumerStub{}, createPartitionConsumersWithClosure(singleMessage), nil
				},
				options: []OptionFunc{Retries(2), RetryWait(time.Nanosecond)},
			},
			expectedMessages: expectedSingleMessage,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg, err := kafka.DefaultConsumerSaramaConfig("test", true)
			require.NoError(t, err)
			component, err := New("foo", []string{"localhost"}, "topic", tt.args.procMock.record, cfg, tt.args.options...)
			require.NoError(t, err)
			component.partitionConsumerFactory = tt.args.partitionConsumerFactory
			err = component.Run(tt.args.ctx)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
				tt.args.procMock.assert(t, tt.expectedMessages)
			}
		})
	}
}

type clientStub struct{}

func (c clientStub) Config() *sarama.Config {
	return nil
}

func (c clientStub) Controller() (*sarama.Broker, error) {
	return nil, nil
}

func (c clientStub) RefreshController() (*sarama.Broker, error) {
	return nil, nil
}

func (c clientStub) Brokers() []*sarama.Broker {
	return nil
}

func (c clientStub) Broker(brokerID int32) (*sarama.Broker, error) {
	return nil, nil
}

func (c clientStub) Topics() ([]string, error) {
	return nil, nil
}

func (c clientStub) Partitions(topic string) ([]int32, error) {
	return nil, nil
}

func (c clientStub) WritablePartitions(topic string) ([]int32, error) {
	return nil, nil
}

func (c clientStub) Leader(topic string, partitionID int32) (*sarama.Broker, error) {
	return nil, nil
}

func (c clientStub) Replicas(topic string, partitionID int32) ([]int32, error) {
	return nil, nil
}

func (c clientStub) InSyncReplicas(topic string, partitionID int32) ([]int32, error) {
	return nil, nil
}

func (c clientStub) OfflineReplicas(topic string, partitionID int32) ([]int32, error) {
	return nil, nil
}

func (c clientStub) RefreshBrokers(addrs []string) error {
	return nil
}

func (c clientStub) RefreshMetadata(topics ...string) error {
	return nil
}

func (c clientStub) GetOffset(topic string, partitionID int32, time int64) (int64, error) {
	return 0, nil
}

func (c clientStub) Coordinator(consumerGroup string) (*sarama.Broker, error) {
	return nil, nil
}

func (c clientStub) RefreshCoordinator(consumerGroup string) error {
	return nil
}

func (c clientStub) InitProducerID() (*sarama.InitProducerIDResponse, error) {
	return nil, nil
}

func (c clientStub) Close() error {
	return nil
}

func (c clientStub) Closed() bool {
	return false
}

type consumerStub struct{}

func (c consumerStub) Topics() ([]string, error) {
	return nil, nil
}

func (c consumerStub) Partitions(topic string) ([]int32, error) {
	return nil, nil
}

func (c consumerStub) ConsumePartition(topic string, partition int32, offset int64) (sarama.PartitionConsumer, error) {
	return nil, nil
}

func (c consumerStub) HighWaterMarks() map[string]map[int32]int64 {
	return nil
}

func (c consumerStub) Close() error {
	return nil
}
