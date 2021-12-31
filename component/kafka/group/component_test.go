package group

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Parallel()

	saramaCfg := sarama.NewConfig()
	// consumer will commit every batch in a blocking operation
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
		p            kafka.BatchProcessorFunc
		fs           kafka.FailStrategy
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
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, retryWait: 2, batchTimeout: time.Second, fs: kafka.ExitStrategy, saramaCfg: saramaCfg},
			wantErr: false,
		},
		{
			name:    "failed, no sarama config",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, retryWait: 2, batchTimeout: time.Second, fs: kafka.ExitStrategy, saramaCfg: nil},
			wantErr: true,
		},
		{
			name:    "failed, missing name",
			args:    args{name: "", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: kafka.ExitStrategy, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, missing group",
			args:    args{name: "name", group: "", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: kafka.ExitStrategy, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, no brokers",
			args:    args{name: "name", group: "grp", brokers: []string{}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: kafka.ExitStrategy, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, no topics",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{""}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: kafka.ExitStrategy, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, missing processor func",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, batchSize: 1, batchTimeout: time.Second, p: nil, fs: kafka.ExitStrategy, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, invalid fail strategy",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: 2, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, invalid retry retry timeout",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, retryWait: -2, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, invalid batch size",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 0, batchTimeout: time.Second, retryWait: 2, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, invalid batch timeout",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: -2, retryWait: 2, saramaCfg: saramaCfg},
			wantErr: true,
		},
		{
			name:    "failed, no sarama configuration",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 10, batchTimeout: time.Second, retryWait: 2, saramaCfg: nil},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := New(
				tt.args.name,
				tt.args.group,
				tt.args.brokers,
				tt.args.topics,
				tt.args.p,
				tt.args.saramaCfg,
				FailureStrategy(tt.args.fs),
				Retries(tt.args.retries),
				RetryWait(tt.args.retryWait),
				BatchSize(tt.args.batchSize),
				BatchTimeout(tt.args.batchTimeout),
				CommitSync())
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
	mux       sync.Mutex
	execs     int
}

var errProcess = errors.New("PROC ERROR")

func (mp *mockProcessor) Process(batch kafka.Batch) error {
	mp.mux.Lock()
	mp.execs += len(batch.Messages())
	mp.mux.Unlock()
	if mp.errReturn {
		return errProcess
	}
	return nil
}

func (mp *mockProcessor) GetExecs() int {
	mp.mux.Lock()
	defer mp.mux.Unlock()
	return mp.execs
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
	proc   *mockProcessor
	mu     sync.RWMutex
	closed bool
}

func (m *mockConsumerClaim) Messages() <-chan *sarama.ConsumerMessage {
	m.mu.Lock()
	if m.proc.GetExecs() > len(m.ch) && !m.closed {
		close(m.ch)
		m.closed = true
	}
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
		proc         *mockProcessor
		failStrategy kafka.FailStrategy
		batchSize    uint
		expectError  bool
	}{
		{
			name: "success",
			msgs: saramaConsumerMessages(json.Type),
			proc: &mockProcessor{
				errReturn: false,
			},
			failStrategy: kafka.ExitStrategy,
			batchSize:    1,
			expectError:  false,
		},
		{
			name: "success-batched",
			msgs: []*sarama.ConsumerMessage{
				saramaConsumerMessage("1", &sarama.RecordHeader{
					Key:   []byte(encoding.ContentTypeHeader),
					Value: []byte(json.Type),
				}),
				saramaConsumerMessage("2", &sarama.RecordHeader{
					Key:   []byte(encoding.ContentTypeHeader),
					Value: []byte(json.Type),
				}),
				saramaConsumerMessage("3", &sarama.RecordHeader{
					Key:   []byte(encoding.ContentTypeHeader),
					Value: []byte(json.Type),
				}),
			},
			proc: &mockProcessor{
				errReturn: false,
			},
			failStrategy: kafka.ExitStrategy,
			batchSize:    10,
			expectError:  false,
		},
		{
			name: "failure",
			msgs: saramaConsumerMessages("mock"),
			proc: &mockProcessor{
				errReturn: true,
			},
			failStrategy: kafka.ExitStrategy,
			batchSize:    1,
			expectError:  true,
		},
		{
			name: "failure-skip",
			msgs: saramaConsumerMessages("mock"),
			proc: &mockProcessor{
				errReturn: true,
			},
			failStrategy: kafka.SkipStrategy,
			batchSize:    1,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			h := newConsumerHandler(ctx, tt.name, "grp", tt.proc.Process, tt.failStrategy, tt.batchSize,
				10*time.Millisecond, true)

			ch := make(chan *sarama.ConsumerMessage, len(tt.msgs))
			for _, m := range tt.msgs {
				ch <- m
			}
			session := &mockConsumerSession{}
			_ = h.Setup(session)
			err := h.ConsumeClaim(session, &mockConsumerClaim{ch: ch, proc: tt.proc})
			_ = h.Cleanup(session)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, len(tt.msgs), tt.proc.GetExecs())
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

func Test_getCorrelationID(t *testing.T) {
	corID := uuid.New().String()
	got := getCorrelationID([]*sarama.RecordHeader{
		{
			Key:   []byte(correlation.HeaderID),
			Value: []byte(corID),
		},
	})
	assert.Equal(t, corID, got)

	emptyCorID := ""
	got = getCorrelationID([]*sarama.RecordHeader{
		{
			Key:   []byte(correlation.HeaderID),
			Value: []byte(emptyCorID),
		},
	})
	assert.NotEqual(t, emptyCorID, got)
}
