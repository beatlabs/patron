//go:build !integration

package kafka

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/beatlabs/patron/correlation"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/plugin/kotel"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/goleak"
)

var (
	tracePublisher *tracesdk.TracerProvider
	traceExporter  = tracetest.NewInMemoryExporter()
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("google.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/metric.(*PeriodicReader).run"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"),
	)
}

func init() {
	if err := os.Setenv("OTEL_BSP_SCHEDULE_DELAY", "100"); err != nil {
		panic(err)
	}

	tracePublisher = patrontrace.Setup("test", nil, traceExporter)
}

func TestNew(t *testing.T) {
	t.Parallel()

	opts := []kgo.Opt{}

	proc := mockProcessor{}
	type args struct {
		name         string
		group        string
		brokers      []string
		topics       []string
		p            ProcessorFunc
		fs           FailStrategy
		retries      uint32
		retryWait    time.Duration
		batchSize    uint
		batchTimeout time.Duration
		opts         []kgo.Opt
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "success",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, retryWait: 2 * time.Second, batchTimeout: time.Second, fs: ExitStrategy, opts: opts},
			wantErr: false,
		},
		{
			name:    "failed, missing name",
			args:    args{name: "", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: ExitStrategy, opts: opts},
			wantErr: true,
		},
		{
			name:    "failed, missing group",
			args:    args{name: "name", group: "", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: ExitStrategy, opts: opts},
			wantErr: true,
		},
		{
			name:    "failed, no brokers",
			args:    args{name: "name", group: "grp", brokers: []string{}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: ExitStrategy, opts: opts},
			wantErr: true,
		},
		{
			name:    "failed, no topics",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{""}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: ExitStrategy, opts: opts},
			wantErr: true,
		},
		{
			name:    "failed, missing processor func",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, batchSize: 1, batchTimeout: time.Second, p: nil, fs: ExitStrategy, opts: opts},
			wantErr: true,
		},
		{
			name:    "failed, invalid fail strategy",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, fs: 2, opts: opts},
			wantErr: true,
		},
		{
			name:    "failed, invalid retry timeout",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: time.Second, retryWait: -2, opts: opts},
			wantErr: true,
		},
		{
			name:    "failed, invalid batch size",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 0, batchTimeout: time.Second, retryWait: 2, opts: opts},
			wantErr: true,
		},
		{
			name:    "failed, invalid batch timeout",
			args:    args{name: "name", group: "grp", brokers: []string{"localhost:9092"}, topics: []string{"topicone"}, p: proc.Process, batchSize: 1, batchTimeout: -2, retryWait: 2, opts: opts},
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
				tt.args.opts,
				WithFailureStrategy(tt.args.fs),
				WithRetries(tt.args.retries),
				WithRetryWait(tt.args.retryWait),
				WithBatchSize(tt.args.batchSize),
				WithBatchTimeout(tt.args.batchTimeout),
				WithManualCommit())
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
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

func (mp *mockProcessor) Process(_ context.Context, records []*kgo.Record) error {
	mp.mux.Lock()
	mp.execs += len(records)
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

func TestConsumerHandler_Flush(t *testing.T) {
	tests := []struct {
		name                      string
		records                   []*kgo.Record
		proc                      *mockProcessor
		failStrategy              FailStrategy
		batchSize                 uint
		batchMessageDeduplication bool
		expectError               bool
		expectedProcessExecutions int
	}{
		{
			name:                      "empty buffer",
			records:                   nil,
			proc:                      &mockProcessor{errReturn: false},
			failStrategy:              ExitStrategy,
			batchSize:                 1,
			expectError:               false,
			expectedProcessExecutions: 0,
		},
		{
			name: "success single message",
			records: []*kgo.Record{
				{Topic: "TEST_TOPIC", Partition: 0, Key: []byte("key"), Value: []byte("value"), Offset: 0},
			},
			proc:                      &mockProcessor{errReturn: false},
			failStrategy:              ExitStrategy,
			batchSize:                 1,
			expectError:               false,
			expectedProcessExecutions: 1,
		},
		{
			name: "success batch",
			records: []*kgo.Record{
				{Topic: "TEST_TOPIC", Partition: 0, Key: []byte("1"), Value: []byte("v1"), Offset: 0},
				{Topic: "TEST_TOPIC", Partition: 0, Key: []byte("2"), Value: []byte("v2"), Offset: 1},
				{Topic: "TEST_TOPIC", Partition: 0, Key: []byte("3"), Value: []byte("v3"), Offset: 2},
			},
			proc:                      &mockProcessor{errReturn: false},
			failStrategy:              ExitStrategy,
			batchSize:                 10,
			expectError:               false,
			expectedProcessExecutions: 3,
		},
		{
			name: "failure exit strategy",
			records: []*kgo.Record{
				{Topic: "TEST_TOPIC", Partition: 0, Key: []byte("key"), Value: []byte("value"), Offset: 0},
			},
			proc:                      &mockProcessor{errReturn: true},
			failStrategy:              ExitStrategy,
			batchSize:                 1,
			expectError:               true,
			expectedProcessExecutions: 1,
		},
		{
			name: "failure skip strategy",
			records: []*kgo.Record{
				{Topic: "TEST_TOPIC", Partition: 0, Key: []byte("key"), Value: []byte("value"), Offset: 0},
			},
			proc:                      &mockProcessor{errReturn: true},
			failStrategy:              SkipStrategy,
			batchSize:                 1,
			expectError:               false,
			expectedProcessExecutions: 1,
		},
		{
			name: "deduplicates messages",
			records: []*kgo.Record{
				{Topic: "TEST_TOPIC", Key: []byte("1"), Value: []byte("v1.1")},
				{Topic: "TEST_TOPIC", Key: []byte("1"), Value: []byte("v1.2")},
				{Topic: "TEST_TOPIC", Key: []byte("2"), Value: []byte("v2.1")},
			},
			proc:                      &mockProcessor{errReturn: false},
			failStrategy:              SkipStrategy,
			batchSize:                 10,
			batchMessageDeduplication: true,
			expectError:               false,
			expectedProcessExecutions: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tracer := kotel.NewTracer()
			handler := newConsumerHandler(ctx, tt.name, "grp", tracer, tt.proc.Process, tt.failStrategy, tt.batchSize,
				10*time.Millisecond, false, tt.batchMessageDeduplication)

			handler.recBuf = append(handler.recBuf, tt.records...)

			err := handler.flush(nil)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectedProcessExecutions, tt.proc.GetExecs())
		})
	}
}

func TestConsumerHandler_FlushWithManualCommit(t *testing.T) {
	ctx := context.Background()
	tracer := kotel.NewTracer()
	proc := &mockProcessor{errReturn: false}

	handler := newConsumerHandler(ctx, "test", "grp", tracer, proc.Process, ExitStrategy, 10,
		10*time.Millisecond, true, false)

	handler.recBuf = append(handler.recBuf, &kgo.Record{
		Topic: "TEST_TOPIC", Partition: 0, Key: []byte("key"), Value: []byte("value"), Offset: 0,
	})

	// With nil client, flush succeeds (client nil guard) but the manualCommit path
	// is verified by checking that processing occurred and the handler flag is set.
	err := handler.flush(nil)
	require.NoError(t, err)
	assert.Equal(t, 1, proc.GetExecs())
	assert.True(t, handler.processedMessages)
	assert.True(t, handler.manualCommit)
}

func TestConsumerHandler_BatchTimeoutFlush(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	tracer := kotel.NewTracer()
	proc := &mockProcessor{errReturn: false}

	handler := newConsumerHandler(ctx, "test", "grp", tracer, proc.Process, ExitStrategy, 100,
		1*time.Millisecond, false, false)

	handler.recBuf = append(handler.recBuf, &kgo.Record{
		Topic: "TEST_TOPIC", Partition: 0, Key: []byte("key"), Value: []byte("value"), Offset: 0,
	})

	time.Sleep(5 * time.Millisecond)

	select {
	case <-handler.ticker.C:
		handler.mu.Lock()
		err := handler.flush(nil)
		handler.mu.Unlock()
		require.NoError(t, err)
	case <-time.After(50 * time.Millisecond):
		t.Fatal("ticker did not fire within expected time")
	}

	assert.Equal(t, 1, proc.GetExecs())
	assert.Empty(t, handler.recBuf)

	cancel()
	handler.ticker.Stop()
}

func TestExecuteFailureStrategy_Unknown(t *testing.T) {
	ctx := context.Background()
	tracer := kotel.NewTracer()
	proc := &mockProcessor{errReturn: false}

	handler := newConsumerHandler(ctx, "test", "grp", tracer, proc.Process, FailStrategy(99), 1,
		10*time.Millisecond, false, false)

	_, sp := tracer.WithProcessSpan(&kgo.Record{Topic: "test"})
	recordSpans := []recordSpan{
		{record: &kgo.Record{Topic: "test", Key: []byte("key"), Value: []byte("value")}, span: sp},
	}

	err := handler.executeFailureStrategy(recordSpans, errors.New("processing error"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown failure strategy")
}

func TestConsumerErrorsInc(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		consumerErrorsInc(context.Background(), "test-consumer")
	})
}

func Test_getCorrelationID(t *testing.T) {
	t.Parallel()

	t.Run("header found with value", func(t *testing.T) {
		t.Parallel()

		corID := uuid.New().String()
		got := getCorrelationID([]kgo.RecordHeader{
			{
				Key:   correlation.HeaderID,
				Value: []byte(corID),
			},
		})
		assert.Equal(t, corID, got)
	})

	t.Run("header found with empty value", func(t *testing.T) {
		t.Parallel()

		got := getCorrelationID([]kgo.RecordHeader{
			{
				Key:   correlation.HeaderID,
				Value: []byte(""),
			},
		})
		assert.NotEmpty(t, got)
		_, err := uuid.Parse(got)
		assert.NoError(t, err)
	})

	t.Run("header missing entirely", func(t *testing.T) {
		t.Parallel()

		got := getCorrelationID([]kgo.RecordHeader{
			{Key: "some-other-header", Value: []byte("value")},
		})
		assert.NotEmpty(t, got)
		_, err := uuid.Parse(got)
		assert.NoError(t, err)
	})

	t.Run("empty headers slice", func(t *testing.T) {
		t.Parallel()

		got := getCorrelationID(nil)
		assert.NotEmpty(t, got)
		_, err := uuid.Parse(got)
		assert.NoError(t, err)
	})
}

func Test_deduplicateRecords(t *testing.T) {
	original := []*kgo.Record{
		{Key: []byte("k1"), Value: []byte("v1.1")},
		{Key: []byte("k1"), Value: []byte("v1.2")},
		{Key: []byte("k2"), Value: []byte("v2.1")},
		{Key: []byte("k2"), Value: []byte("v2.2")},
		{Key: []byte("k1"), Value: []byte("v1.3")},
	}

	cleaned := deduplicateRecords(original)

	assert.Len(t, cleaned, 2)
	assert.Equal(t, []byte("k2"), cleaned[0].Key)
	assert.Equal(t, []byte("v2.2"), cleaned[0].Value)
	assert.Equal(t, []byte("k1"), cleaned[1].Key)
	assert.Equal(t, []byte("v1.3"), cleaned[1].Value)
}
