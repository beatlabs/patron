package sqs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	sp := stubProcessor{t: t}

	type args struct {
		name      string
		queueName string
		sqsAPI    sqsiface.SQSAPI
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
				oo:        []OptionFunc{Retries(5)},
			},
		},
		"missing name": {
			args: args{
				name:      "",
				queueName: "queueName",
				sqsAPI:    &stubSQSAPI{},
				proc:      sp.process,
				oo:        []OptionFunc{Retries(5)},
			},
			expectedErr: "component name is empty",
		},
		"missing queue name": {
			args: args{
				name:      "name",
				queueName: "",
				sqsAPI:    &stubSQSAPI{},
				proc:      sp.process,
				oo:        []OptionFunc{Retries(5)},
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
				oo:   []OptionFunc{Retries(5)},
			},
			expectedErr: "failed to get queue URL: QUEUE URL ERROR",
		},
		"missing queue SQS API": {
			args: args{
				name:      "name",
				queueName: "queueName",
				sqsAPI:    nil,
				proc:      sp.process,
				oo:        []OptionFunc{Retries(5)},
			},
			expectedErr: "SQS API is nil",
		},
		"missing process function": {
			args: args{
				name:      "name",
				queueName: "queueName",
				sqsAPI:    &stubSQSAPI{},
				proc:      nil,
				oo:        []OptionFunc{Retries(5)},
			},
			expectedErr: "process function is nil",
		},
		"retry option fails": {
			args: args{
				name:      "name",
				queueName: "queueName",
				sqsAPI:    &stubSQSAPI{},
				proc:      sp.process,
				oo:        []OptionFunc{RetryWait(-1 * time.Second)},
			},
			expectedErr: "retry wait time should be a positive number",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
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
	defer mockTracer.Reset()
	sp := stubProcessor{t: t}

	sqsAPI := stubSQSAPI{
		succeededMessage: createMessage(nil, "1"),
		failedMessage:    createMessage(nil, "2"),
	}
	cmp, err := New("name", queueName, sqsAPI, sp.process, QueueStatsInterval(10*time.Millisecond))
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
	assert.True(t, len(mockTracer.FinishedSpans()) > 0)
}

func TestComponent_RunEvenIfStatsFail_Success(t *testing.T) {
	defer mockTracer.Reset()
	sp := stubProcessor{t: t}

	sqsAPI := stubSQSAPI{
		succeededMessage:                 createMessage(nil, "1"),
		failedMessage:                    createMessage(nil, "2"),
		getQueueAttributesWithContextErr: errors.New("STATS FAIL"),
	}
	cmp, err := New("name", queueName, sqsAPI, sp.process, QueueStatsInterval(10*time.Millisecond))
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
	assert.True(t, len(mockTracer.FinishedSpans()) > 0)
}

func TestComponent_Run_Error(t *testing.T) {
	defer mockTracer.Reset()
	sp := stubProcessor{t: t}

	sqsAPI := stubSQSAPI{
		receiveMessageWithContextErr: errors.New("FAILED FETCH"),
		succeededMessage:             createMessage(nil, "1"),
		failedMessage:                createMessage(nil, "2"),
	}
	cmp, err := New("name", queueName, sqsAPI, sp.process, Retries(2), RetryWait(10*time.Millisecond))
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
