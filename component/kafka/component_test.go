package kafka

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
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

type proxyBuilder struct {
	proc         mockProcessor
	fs           FailStrategy
	retries      int
	retryWait    time.Duration
	syncCommit   bool
	batchSize    uint
	batchTimeout time.Duration
	saramaCfg    *sarama.Config
}

func run(ctx context.Context, t *testing.T, mockBroker *sarama.MockBroker, topic string, builder *proxyBuilder) error {
	cmpb := New("test", "test-grp", []string{mockBroker.Addr()}, []string{topic}, builder.proc.Process).
		WithFailureStrategy(builder.fs).
		WithRetries(uint(builder.retries)).
		WithRetryWait(builder.retryWait).
		WithBatching(builder.batchSize, builder.batchTimeout).
		WithSaramaConfig(builder.saramaCfg)

	if builder.syncCommit {
		cmpb.WithSyncCommit()
	}

	cmp, err := cmpb.Create()
	assert.NoError(t, err)
	return cmp.Run(ctx)
}

// TestRun_ReturnsError expects a consumer consume Error
func TestRun_ReturnsErrorOnExitStrategy(t *testing.T) {
	topic := "topicone"
	saramaCfg, err := defaultSaramaConfig("test")
	assert.NoError(t, err)
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	builder := proxyBuilder{
		proc:         mockProcessor{errReturn: true, t: t},
		fs:           ExitStrategy,
		syncCommit:   true,
		batchSize:    1,
		batchTimeout: time.Second,
		saramaCfg:    saramaCfg,
	}
	broker := sarama.NewMockBroker(t, 0)
	defer broker.Close()
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(broker.Addr(), broker.BrokerID()).
			SetLeader(topic, 0, broker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset(topic, 0, sarama.OffsetNewest, 2).
			SetOffset(topic, 0, sarama.OffsetOldest, 0),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1).
			SetMessage(topic, 0, 0, sarama.StringEncoder("{}")),
	})

	assert.NoError(t, err)
	err = run(context.Background(), t, broker, topic, &builder)
	assert.True(t, errors.Is(err, errProcess))
	assert.Equal(t, 0, builder.proc.execs)
}

type mockProcessor struct {
	errReturn bool
	mux       sync.Mutex
	execs     int
	t         *testing.T
}

var errProcess = errors.New("PROC ERROR")

func (mp *mockProcessor) Process(context.Context, []MessageWrapper) error {
	mp.t.Log("addddd")
	mp.mux.Lock()
	mp.execs++
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
