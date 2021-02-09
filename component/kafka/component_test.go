package kafka

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	saramaCfg := sarama.NewConfig()
	// consumer will use WithSyncCommit so will manually commit offsets
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = false
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaCfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky
	saramaCfg.Net.DialTimeout = 15 * time.Second
	saramaCfg.Version = sarama.V2_6_0_0

	proc := mockProcessor{}
	type args struct {
		name         string
		group        string
		brokers      []string
		topics       []string
		p            BatchProcessorFunc
		fs           FailStrategy
		retries      uint
		retryWait    time.Duration
		batchSize    uint
		batchTimeout time.Duration
		saramaCfg    *sarama.Config
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "success",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: ExitStrategy, saramaCfg: saramaCfg},
			wantErr: false,
		},
		{
			name:    "success, no sarama config",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: ExitStrategy, saramaCfg: nil},
			wantErr: false,
		},
		{
			name:    "failed, missing name",
			args:    args{name: "", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: ExitStrategy, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, missing group",
			args:    args{name: "name", group: "", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: ExitStrategy, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, no brokers",
			args:    args{name: "name", group: "grp", brokers: []string{}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: ExitStrategy, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, no topics",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{""}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: ExitStrategy, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, missing processor func",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, batchSize: 1, batchTimeout: time.Second, p: nil, fs: ExitStrategy, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, invalid fail strategy",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: 2, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, invalid retry retry timeout",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, retryWait: -2},
			wantErr: true,
		},
		{
			name:    "failed, invalid batch size",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 0, batchTimeout: time.Second, retryWait: 2},
			wantErr: true,
		},
		{
			name:    "failed, invalid batch timeout",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: -2, retryWait: 2},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.name, tt.args.group, tt.args.brokers, tt.args.topics, tt.args.p).
				WithFailureStrategy(tt.args.fs).
				WithRetries(tt.args.retries).
				WithRetryWait(tt.args.retryWait).
				WithBatching(tt.args.batchSize, tt.args.batchTimeout).
				WithSyncCommit().
				WithSaramaConfig(tt.args.saramaCfg).
				Create()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

type mockProcessor struct {
	errReturn bool
}

var errProcess = errors.New("PROC ERROR")

func (mp *mockProcessor) Process(context.Context, []MessageWrapper) error {
	if mp.errReturn {
		return errProcess
	}
	return nil
}

func Test_mapHeader(t *testing.T) {
	mp := mapHeader([]*sarama.RecordHeader{
		{
			Key:   []byte("X-HEADER-1"),
			Value: []byte("1"),
		},
		{
			Key:   []byte("X-HEADER-2"),
			Value: []byte("2"),
		},
	})
	assert.Equal(t, "1", mp["X-HEADER-1"])
	assert.Equal(t, "2", mp["X-HEADER-2"])
	_, ok := mp["X-HEADER-3"]
	assert.False(t, ok)
}

type mockConsumerClaim struct {
	ch     chan *sarama.ConsumerMessage
	cancel context.CancelFunc
	mu     sync.RWMutex
	closed bool
}

func (m *mockConsumerClaim) Messages() <-chan *sarama.ConsumerMessage {
	m.mu.Lock()
	if m.closed {
		m.cancel()
	} else {
		close(m.ch)
	}
	m.closed = true
	m.mu.Unlock()
	return m.ch
}
func (m *mockConsumerClaim) Topic() string              { return "" }
func (m *mockConsumerClaim) Partition() int32           { return 0 }
func (m *mockConsumerClaim) InitialOffset() int64       { return 0 }
func (m *mockConsumerClaim) HighWaterMarkOffset() int64 { return 1 }

type mockConsumerSession struct{}

func (m *mockConsumerSession) Claims() map[string][]int32 { return nil }
func (m *mockConsumerSession) MemberID() string           { return "" }
func (m *mockConsumerSession) GenerationID() int32        { return 0 }
func (m *mockConsumerSession) MarkOffset(string, int32, int64, string) {
}
func (m *mockConsumerSession) Commit() {}
func (m *mockConsumerSession) ResetOffset(string, int32, int64, string) {
}
func (m *mockConsumerSession) MarkMessage(*sarama.ConsumerMessage, string) {}
func (m *mockConsumerSession) Context() context.Context                    { return context.Background() }

func TestHandler_ConsumeClaim(t *testing.T) {

	tests := []struct {
		name         string
		msgs         []*sarama.ConsumerMessage
		proc         BatchProcessorFunc
		failStrategy FailStrategy
		expectError  bool
	}{
		{
			name: "success",
			msgs: saramaConsumerMessages(json.Type),
			proc: func(context.Context, []MessageWrapper) error {
				return nil
			},
			failStrategy: ExitStrategy,
			expectError:  false,
		},
		{
			name: "failure",
			msgs: saramaConsumerMessages("mock"),
			proc: func(context.Context, []MessageWrapper) error {
				return errors.New("mock-error")
			},
			failStrategy: ExitStrategy,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			h := newConsumerHandler(ctx, cancel, tt.name, "grp", tt.proc, tt.failStrategy, 1,
				time.Second, 0, 0, true)

			ch := make(chan *sarama.ConsumerMessage, len(tt.msgs))
			for _, m := range tt.msgs {
				ch <- m
			}
			err := h.ConsumeClaim(&mockConsumerSession{}, &mockConsumerClaim{ch: ch, cancel: cancel})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func saramaConsumerMessages(ct string) []*sarama.ConsumerMessage {
	return []*sarama.ConsumerMessage{
		saramaConsumerMessage("value", &sarama.RecordHeader{
			Key:   []byte(encoding.ContentTypeHeader),
			Value: []byte(ct),
		}),
	}
}

func saramaConsumerMessage(value string, header *sarama.RecordHeader) *sarama.ConsumerMessage {
	return versionedConsumerMessage(value, header, 0)
}

func versionedConsumerMessage(value string, header *sarama.RecordHeader, version uint8) *sarama.ConsumerMessage {

	bytes := []byte(value)

	if version > 0 {
		bytes = append([]byte{version}, bytes...)
	}

	return &sarama.ConsumerMessage{
		Topic:          "TEST_TOPIC",
		Partition:      0,
		Key:            []byte("key"),
		Value:          bytes,
		Offset:         0,
		Timestamp:      time.Now(),
		BlockTimestamp: time.Now(),
		Headers:        []*sarama.RecordHeader{header},
	}
}
