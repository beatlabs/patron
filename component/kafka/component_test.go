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
		p            BatchProcessorFunc
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
				WithCommitSync())
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

func (mp *mockProcessor) Process(batch Batch) error {
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

func Test_getCorrelationID(t *testing.T) {
	corID := uuid.New().String()
	got := getCorrelationID([]kgo.RecordHeader{
		{
			Key:   correlation.HeaderID,
			Value: []byte(corID),
		},
	})
	assert.Equal(t, corID, got)

	emptyCorID := ""
	got = getCorrelationID([]kgo.RecordHeader{
		{
			Key:   correlation.HeaderID,
			Value: []byte(emptyCorID),
		},
	})
	assert.NotEqual(t, emptyCorID, got)
}

func Test_deduplicateMessages(t *testing.T) {
	message := func(key, val string) Message {
		return NewMessage(
			context.Background(),
			nil,
			&kgo.Record{Key: []byte(key), Value: []byte(val)})
	}

	// Given
	original := []Message{
		message("k1", "v1.1"),
		message("k1", "v1.2"),
		message("k2", "v2.1"),
		message("k2", "v2.2"),
		message("k1", "v1.3"),
	}

	cleaned := deduplicateMessages(original)

	assert.Len(t, cleaned, 2)
	// Verify ordering is preserved based on the position of the winning (last) message:
	// k2's last occurrence is at index 3, k1's last occurrence is at index 4
	assert.Equal(t, []byte("k2"), cleaned[0].Record().Key)
	assert.Equal(t, []byte("v2.2"), cleaned[0].Record().Value)
	assert.Equal(t, []byte("k1"), cleaned[1].Record().Key)
	assert.Equal(t, []byte("v1.3"), cleaned[1].Record().Value)
}
