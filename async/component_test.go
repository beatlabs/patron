package async

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	proc := mockProcessor{}
	type args struct {
		name      string
		p         ProcessorFunc
		cf        ConsumerFactory
		fs        FailStrategy
		retries   uint
		retryWait time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "success",
			args:    args{name: "name", p: proc.Process, cf: &mockConsumerFactory{}, fs: NackExitStrategy},
			wantErr: false,
		},
		{
			name:    "failed, missing name",
			args:    args{name: "", p: proc.Process, cf: &mockConsumerFactory{}, fs: NackExitStrategy},
			wantErr: true,
		},
		{
			name:    "failed, missing processor func",
			args:    args{name: "name", p: nil, cf: &mockConsumerFactory{}, fs: NackExitStrategy},
			wantErr: true,
		},
		{
			name:    "failed, missing consumer",
			args:    args{name: "name", p: proc.Process, cf: nil, fs: NackExitStrategy},
			wantErr: true,
		},
		{
			name:    "failed, invalid fail strategy",
			args:    args{name: "name", p: proc.Process, cf: &mockConsumerFactory{}, fs: 3},
			wantErr: true,
		},
		{
			name:    "failed, invalid retry retry timeout",
			args:    args{name: "name", p: proc.Process, cf: &mockConsumerFactory{}, retryWait: -2},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.name).
				WithProcessor(tt.args.p).
				WithConsumerFactory(tt.args.cf).
				WithFailureStrategy(tt.args.fs).
				WithRetries(tt.args.retries).
				WithRetryWait(tt.args.retryWait).
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
	proc      mockProcessor
	cnr       mockConsumer
	cf        ConsumerFactory
	fs        FailStrategy
	retries   int
	retryWait time.Duration
}

func run(t *testing.T, ctx context.Context, builder *proxyBuilder) error {

	if builder.cf == nil {
		builder.cf = &mockConsumerFactory{c: &builder.cnr}
	}

	cmp, err := New("test").
		WithProcessor(builder.proc.Process).
		WithConsumerFactory(builder.cf).
		WithFailureStrategy(builder.fs).
		WithRetries(uint(builder.retries)).
		WithRetryWait(builder.retryWait).
		Create()
	assert.NoError(t, err)
	return cmp.Run(ctx)
}

// TestRun_ReturnsError expects a consumer consume Error
func TestRun_ReturnsError(t *testing.T) {

	builder := proxyBuilder{
		cnr: mockConsumer{consumeError: true},
	}
	err := run(t, context.Background(), &builder)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), consumerError.Error()))
	assert.Equal(t, 0, builder.proc.execs)

}

// TestRun_WithCancel_CloseError expects a consumer closing error
func TestRun_WithCancel_CloseError(t *testing.T) {

	builder := proxyBuilder{
		cnr: mockConsumer{clsError: true},
	}

	ctx, cnl := context.WithCancel(context.Background())
	cnl()

	err := run(t, ctx, &builder)

	assert.Error(t, err)
	assert.Equal(t, consumerCloseError, err)
	assert.Equal(t, 0, builder.proc.execs)

}

// TestRun_Process_Error_NackExitStrategy expects a PROC ERROR
// from an error producing processor
// which will cause the component to return with an error
// due to the Nack FailureStrategy
func TestRun_Process_Error_NackExitStrategy(t *testing.T) {

	builder := proxyBuilder{
		proc: mockProcessor{retError: true},
		cnr: mockConsumer{
			chMsg: make(chan Message, 10),
			chErr: make(chan error, 10),
		},
	}

	ctx := context.Background()
	builder.cnr.chMsg <- &mockMessage{ctx: ctx}

	err := run(t, ctx, &builder)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), processError.Error()))
	assert.Equal(t, 1, builder.proc.execs)

}

// TestRun_Process_Error_NackStrategy expects a PROC ERROR
// from an error producing processor
// but due to the Nack FailureStrategy, it will continue processing other messages
func TestRun_Process_Error_NackStrategy(t *testing.T) {

	builder := proxyBuilder{
		proc: mockProcessor{retError: true},
		cnr: mockConsumer{
			chMsg: make(chan Message, 10),
			chErr: make(chan error, 10),
		},
		fs: NackStrategy,
	}

	ctx, cnl := context.WithCancel(context.Background())
	builder.cnr.chMsg <- &mockMessage{ctx: ctx}
	ch := make(chan bool)
	go func() {
		assert.NoError(t, run(t, ctx, &builder))
		ch <- true
	}()
	time.Sleep(10 * time.Millisecond)
	cnl()

	select {
	case _, ok := <-builder.cnr.chErr:
		if ok {
			assert.Fail(t, "we dont expect an error , given our nack failure strategy setup")
		}
	default:
		assert.True(t, <-ch)
	}
	assert.Equal(t, 1, builder.proc.execs)

}

// TestRun_ProcessError_WithNackError expects a PROC ERROR
// from an error producing processor
// but also a Nack Error from the message
// This will cause the component to stop execution, as it cannot execute the Nack failure strategy
func TestRun_ProcessError_WithNackError(t *testing.T) {

	builder := proxyBuilder{
		proc: mockProcessor{retError: true},
		cnr: mockConsumer{
			chMsg: make(chan Message, 10),
			chErr: make(chan error, 10),
		},
		fs: NackStrategy,
	}

	ctx := context.Background()
	builder.cnr.chMsg <- &mockMessage{ctx: ctx, nackError: true}

	err := run(t, ctx, &builder)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), nackError.Error()))
	assert.Equal(t, 1, builder.proc.execs)

}

// TestRun_Process_Error_AckStrategy expects a PROC ERROR
// from an error producing processor
// but due to the Ack FailureStrategy, it will continue processing other messages
func TestRun_Process_Error_AckStrategy(t *testing.T) {

	builder := proxyBuilder{
		proc: mockProcessor{retError: true},
		cnr: mockConsumer{
			chMsg: make(chan Message, 10),
			chErr: make(chan error, 10),
		},
		fs: NackStrategy,
	}

	ctx, cnl := context.WithCancel(context.Background())
	builder.cnr.chMsg <- &mockMessage{ctx: ctx}
	ch := make(chan bool)
	go func() {
		assert.NoError(t, run(t, ctx, &builder))
		ch <- true
	}()
	time.Sleep(10 * time.Millisecond)
	cnl()

	select {
	case _, ok := <-builder.cnr.chErr:
		if ok {
			assert.Fail(t, "we dont expect an error , given our ack failure strategy setup")
		}
	default:
		assert.True(t, <-ch)
	}
	assert.Equal(t, 1, builder.proc.execs)

}

// TestRun_ProcessError_WithAckError expects a PROC ERROR
// from an error producing processor
// but also an Ack Error from the message
// This will cause the component to stop execution, as it cannot execute the Nack failure strategy
func TestRun_ProcessError_WithAckError(t *testing.T) {

	builder := proxyBuilder{
		proc: mockProcessor{retError: true},
		cnr: mockConsumer{
			chMsg: make(chan Message, 10),
			chErr: make(chan error, 10),
		},
		fs: AckStrategy,
	}

	ctx := context.Background()
	builder.cnr.chMsg <- &mockMessage{ctx: ctx, ackError: true}

	err := run(t, ctx, &builder)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), ackError.Error()))
	assert.Equal(t, 1, builder.proc.execs)

}

// TestRun_MessageAckError expects an ack error from the message acknowledgement
// it will break the execution of the component due to the default NackExit failure strategy
func TestRun_MessageAckError(t *testing.T) {

	builder := proxyBuilder{
		cnr: mockConsumer{
			chMsg: make(chan Message, 10),
			chErr: make(chan error, 10),
		},
	}

	ctx := context.Background()
	builder.cnr.chMsg <- &mockMessage{ctx: ctx, ackError: true}
	err := run(t, ctx, &builder)

	assert.Error(t, err)
	assert.Equal(t, ackError, err)
	assert.Equal(t, 1, builder.proc.execs)

}

// TestRun_ConsumeError will break the component execution,
// when an error is injected into the consumers error channel
// while using the default NackExit Failure Strategy
func TestRun_ConsumeError(t *testing.T) {

	builder := proxyBuilder{
		cnr: mockConsumer{
			chMsg: make(chan Message, 10),
			chErr: make(chan error, 10),
		},
	}

	ctx := context.Background()
	builder.cnr.chErr <- consumerError
	err := run(t, ctx, &builder)

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), consumerError.Error()))
	assert.Equal(t, 0, builder.proc.execs)

}

// TestRun_ConsumeError_WithRetry will retry the specified amount of times
// before exiting the execution
func TestRun_ConsumeError_WithRetry(t *testing.T) {

	retries := 3
	cf := &mockConsumerFactory{retErr: true}
	builder := proxyBuilder{
		cf:        cf,
		retries:   retries,
		retryWait: 2 * time.Millisecond,
	}

	err := run(t, context.Background(), &builder)

	assert.Error(t, err)
	assert.True(t, retries <= cf.execs)
	assert.Equal(t, 0, builder.proc.execs)

}

// TestRun_ConsumeError_WithRetry_AndContextCancel will retry after a consumer error
// only a small amount fo times, due to the context being canceled as well
func TestRun_ConsumeError_WithRetry_AndContextCancel(t *testing.T) {

	retries := 33
	cf := &mockConsumerFactory{retErr: true}
	builder := proxyBuilder{
		cf:        cf,
		retries:   retries,
		retryWait: 2 * time.Millisecond,
	}

	ctx, cnl := context.WithCancel(context.Background())
	cnl()
	err := run(t, ctx, &builder)

	assert.Error(t, err)
	assert.Equal(t, ctx.Err(), context.Canceled)
	assert.True(t, retries > cf.execs)
	assert.Equal(t, 0, builder.proc.execs)

}

// TestRun_Process_Shutdown verifies the process shuts down on a context cancellation action
func TestRun_Process_Shutdown(t *testing.T) {

	builder := proxyBuilder{
		cnr: mockConsumer{
			chMsg: make(chan Message, 10),
			chErr: make(chan error, 10),
		},
	}

	builder.cnr.chMsg <- &mockMessage{ctx: context.Background()}
	ch := make(chan bool)
	ctx, cnl := context.WithCancel(context.Background())
	go func() {
		err1 := run(t, ctx, &builder)
		assert.NoError(t, err1)
		ch <- true
	}()
	time.Sleep(10 * time.Millisecond)
	cnl()

	assert.True(t, <-ch)
	assert.Equal(t, 1, builder.proc.execs)

}

// TestRun_Process_Error_InvalidStrategy expects a invalid failure strategy error
// NOTE : we injected the failure strategy after the construction,
// in order to avoid the failure strategy check
func TestRun_Process_Error_InvalidStrategy(t *testing.T) {
	cnr := mockConsumer{
		chMsg: make(chan Message, 10),
		chErr: make(chan error, 10),
	}
	proc := mockProcessor{retError: true}
	cmp, err := New("test").
		WithProcessor(proc.Process).
		WithConsumerFactory(&mockConsumerFactory{c: &cnr}).
		Create()
	assert.NoError(t, err)
	cmp.failStrategy = 4
	ctx := context.Background()
	cnr.chMsg <- &mockMessage{ctx: ctx}
	err = cmp.Run(ctx)
	assert.Error(t, err)
	assert.Equal(t, invalidFSError, err)
	assert.Equal(t, 1, proc.execs)

}

type mockMessage struct {
	ctx       context.Context
	ackError  bool
	nackError bool
}

func (mm *mockMessage) Context() context.Context {
	return mm.ctx
}

// Decode is not called in our tests, because the mockProcessor will ignore the message decoding
func (mm *mockMessage) Decode(v interface{}) error {
	return nil
}

var ackError = errors.New("MESSAGE ACK ERROR")

func (mm *mockMessage) Ack() error {
	if mm.ackError {
		return ackError
	}
	return nil
}

var nackError = errors.New("MESSAGE NACK ERROR")

func (mm *mockMessage) Nack() error {
	if mm.nackError {
		return nackError
	}
	return nil
}

type mockProcessor struct {
	retError bool
	execs    int
}

var processError = errors.New("PROC ERROR")

func (mp *mockProcessor) Process(msg Message) error {
	mp.execs++
	if mp.retError {
		return processError
	}
	return nil
}

type mockConsumerFactory struct {
	c      Consumer
	retErr bool
	execs  int
}

var factoryError = errors.New("FACTORY ERROR")

func (mcf *mockConsumerFactory) Create() (Consumer, error) {
	mcf.execs++
	if mcf.retErr {
		return nil, factoryError
	}
	return mcf.c, nil
}

type mockConsumer struct {
	consumeError bool
	clsError     bool
	chMsg        chan Message
	chErr        chan error
}

func (mc *mockConsumer) SetTimeout(timeout time.Duration) {
}

var consumerError = errors.New("CONSUMER ERROR")

func (mc *mockConsumer) Consume(context.Context) (<-chan Message, <-chan error, error) {
	if mc.consumeError {
		return nil, nil, consumerError
	}
	return mc.chMsg, mc.chErr, nil
}

var consumerCloseError = errors.New("CONSUMER CLOSE ERROR")

func (mc *mockConsumer) Close() error {
	if mc.clsError {
		return consumerCloseError
	}
	return nil
}
