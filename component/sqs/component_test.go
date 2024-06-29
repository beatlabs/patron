package sqs

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

var (
	tracePublisher *tracesdk.TracerProvider
	traceExporter  = tracetest.NewInMemoryExporter()
)

func TestMain(m *testing.M) {
	os.Setenv("OTEL_BSP_SCHEDULE_DELAY", "100")

	tracePublisher = patrontrace.Setup("test", nil, traceExporter)

	os.Exit(m.Run())
}

func TestNew(t *testing.T) {
	t.Parallel()
	sp := stubProcessor{t: t}

	type args struct {
		name      string
		queueName string
		sqsAPI    API
		proc      ProcessorFunc
		oo        []OptionFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{
				name:      "name",
				queueName: "queueName",
				sqsAPI:    &stubSQSAPI{},
				proc:      sp.process,
				oo:        []OptionFunc{WithRetries(5)},
			},
		},
		"missing name": {
			args: args{
				name:      "",
				queueName: "queueName",
				sqsAPI:    &stubSQSAPI{},
				proc:      sp.process,
				oo:        []OptionFunc{WithRetries(5)},
			},
			expectedErr: "component name is empty",
		},
		"missing queue name": {
			args: args{
				name:      "name",
				queueName: "",
				sqsAPI:    &stubSQSAPI{},
				proc:      sp.process,
				oo:        []OptionFunc{WithRetries(5)},
			},
			expectedErr: "queue name is empty",
		},
		"missing queue URL": {
			args: args{
				name:      "name",
				queueName: "queueName",
				sqsAPI: &stubSQSAPI{
					getQueueUrlWithContextErr: errors.New("QUEUE URL ERROR"),
				},
				proc: sp.process,
				oo:   []OptionFunc{WithRetries(5)},
			},
			expectedErr: "failed to get queue URL: QUEUE URL ERROR",
		},
		"missing queue SQS API": {
			args: args{
				name:      "name",
				queueName: "queueName",
				sqsAPI:    nil,
				proc:      sp.process,
				oo:        []OptionFunc{WithRetries(5)},
			},
			expectedErr: "SQS API is nil",
		},
		"missing process function": {
			args: args{
				name:      "name",
				queueName: "queueName",
				sqsAPI:    &stubSQSAPI{},
				proc:      nil,
				oo:        []OptionFunc{WithRetries(5)},
			},
			expectedErr: "process function is nil",
		},
		"retry option fails": {
			args: args{
				name:      "name",
				queueName: "queueName",
				sqsAPI:    &stubSQSAPI{},
				proc:      sp.process,
				oo:        []OptionFunc{WithRetryWait(-1 * time.Second)},
			},
			expectedErr: "retry wait time should be a positive number",
		},
		"success queue owner": {
			args: args{
				name:      "name",
				queueName: "queueName",
				sqsAPI:    &stubSQSAPI{},
				proc:      sp.process,
				oo:        []OptionFunc{WithQueueOwner("10201020")},
			},
		},
		"queue owner fails": {
			args: args{
				name:      "name",
				queueName: "queueName",
				sqsAPI:    &stubSQSAPI{},
				proc:      sp.process,
				oo:        []OptionFunc{WithQueueOwner("")},
			},
			expectedErr: "queue owner should not be empty",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := New(tt.args.name, tt.args.queueName, tt.args.sqsAPI, tt.args.proc, tt.args.oo...)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestComponent_Run_Success(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

	sp := stubProcessor{t: t}

	sqsAPI := newStubSQSAPI()
	sqsAPI.succeededMessage = createMessage(nil, "1")
	sqsAPI.failedMessage = createMessage(nil, "2")

	cmp, err := New("name", queueName, sqsAPI, sp.process, WithQueueStatsInterval(10*time.Millisecond))
	require.NoError(t, err)
	ctx, cnl := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		require.NoError(t, cmp.Run(ctx))
		wg.Done()
	}()

	time.Sleep(1 * time.Second)
	cnl()
	wg.Wait()

	assert.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expectedSuc := createStubSpan("sqs-consumer", "")
	expectedFail := createStubSpan("sqs-consumer", "failed to ACK message")

	got := traceExporter.GetSpans()

	assert.Len(t, got, 2)
	assertSpan(t, expectedSuc, got[0])
	assertSpan(t, expectedFail, got[1])
}

func TestComponent_RunEvenIfStatsFail_Success(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

	sp := stubProcessor{t: t}

	sqsAPI := newStubSQSAPI()
	sqsAPI.succeededMessage = createMessage(nil, "1")
	sqsAPI.failedMessage = createMessage(nil, "2")
	sqsAPI.getQueueAttributesWithContextErr = errors.New("STATS FAIL")

	cmp, err := New("name", queueName, sqsAPI, sp.process, WithQueueStatsInterval(10*time.Millisecond))
	require.NoError(t, err)
	ctx, cnl := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		_ = cmp.Run(ctx)
		wg.Done()
	}()

	time.Sleep(1 * time.Second)
	cnl()
	wg.Wait()

	assert.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expectedSuc := createStubSpan("sqs-consumer", "")
	expectedFail := createStubSpan("sqs-consumer", "failed to ACK message")

	got := traceExporter.GetSpans()

	assert.Len(t, got, 2, "expected 2 spans, got %d", len(got))
	assertSpan(t, expectedSuc, got[0])
	assertSpan(t, expectedFail, got[1])
}

func TestComponent_Run_Error(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

	sp := stubProcessor{t: t}

	sqsAPI := stubSQSAPI{
		receiveMessageWithContextErr: errors.New("FAILED FETCH"),
		succeededMessage:             createMessage(nil, "1"),
		failedMessage:                createMessage(nil, "2"),
	}
	cmp, err := New("name", queueName, sqsAPI, sp.process, WithRetries(2), WithRetryWait(10*time.Millisecond))
	require.NoError(t, err)
	ctx, cnl := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		require.Error(t, cmp.Run(ctx))
		wg.Done()
	}()

	time.Sleep(1 * time.Second)
	cnl()
	wg.Wait()
}

type stubProcessor struct {
	t *testing.T
}

func (sp stubProcessor) process(_ context.Context, b Batch) {
	_, err := b.ACK()
	require.NoError(sp.t, err)
}
