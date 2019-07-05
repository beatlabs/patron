package async

import (
	"context"
	"testing"
	"time"

	"github.com/beatlabs/patron/errors"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	proc := mockProcessor{}
	type args struct {
		name string
		p    ProcessorFunc
		cf   ConsumerFactory
		opt  OptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "success",
			args:    args{name: "name", p: proc.Process, cf: &mockConsumerFactory{}, opt: FailureStrategy(NackExitStrategy)},
			wantErr: false,
		},
		{
			name:    "failed, missing name",
			args:    args{name: "", p: proc.Process, cf: &mockConsumerFactory{}, opt: FailureStrategy(NackExitStrategy)},
			wantErr: true,
		},
		{
			name:    "failed, missing processor func",
			args:    args{name: "name", p: nil, cf: &mockConsumerFactory{}, opt: FailureStrategy(NackExitStrategy)},
			wantErr: true,
		},
		{
			name:    "failed, missing consumer",
			args:    args{name: "name", p: proc.Process, cf: nil, opt: FailureStrategy(NackExitStrategy)},
			wantErr: true,
		},
		{
			name:    "failed, invalid fail strategy",
			args:    args{name: "name", p: proc.Process, cf: &mockConsumerFactory{}, opt: FailureStrategy(3)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.name, tt.args.p, tt.args.cf, tt.args.opt)
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

func TestRun_ReturnsError(t *testing.T) {
	cnr := mockConsumer{consumeError: true}
	proc := mockProcessor{}
	cmp, err := New("test", proc.Process, &mockConsumerFactory{c: &cnr})
	assert.NoError(t, err)
	ctx := context.Background()
	err = cmp.Run(ctx)
	assert.Error(t, err)
}

func TestRun_Process_Error_NackExitStrategy(t *testing.T) {
	cnr := mockConsumer{
		chMsg: make(chan Message, 10),
		chErr: make(chan error, 10),
	}
	proc := mockProcessor{retError: true}
	cmp, err := New("test", proc.Process, &mockConsumerFactory{c: &cnr})
	assert.NoError(t, err)
	ctx := context.Background()
	cnr.chMsg <- &mockMessage{ctx: ctx}
	err = cmp.Run(ctx)
	assert.Error(t, err)
}

func TestRun_Process_Error_NackStrategy(t *testing.T) {
	cnr := mockConsumer{
		chMsg: make(chan Message, 10),
		chErr: make(chan error, 10),
	}
	proc := mockProcessor{retError: true}
	cmp, err := New("test", proc.Process, &mockConsumerFactory{c: &cnr}, FailureStrategy(NackStrategy))
	assert.NoError(t, err)
	ctx, cnl := context.WithCancel(context.Background())
	cnr.chMsg <- &mockMessage{ctx: ctx}
	ch := make(chan bool)
	go func() {
		assert.NoError(t, cmp.Run(ctx))
		ch <- true
	}()
	time.Sleep(10 * time.Millisecond)
	cnl()
	assert.True(t, <-ch)
}

func TestRun_Process_Error_AckStrategy(t *testing.T) {
	cnr := mockConsumer{
		chMsg: make(chan Message, 10),
		chErr: make(chan error, 10),
	}
	proc := mockProcessor{retError: true}
	cmp, err := New("test", proc.Process, &mockConsumerFactory{c: &cnr}, FailureStrategy(AckStrategy))
	assert.NoError(t, err)
	ctx, cnl := context.WithCancel(context.Background())
	cnr.chMsg <- &mockMessage{ctx: ctx}
	ch := make(chan bool)
	go func() {
		assert.NoError(t, cmp.Run(ctx))
		ch <- true
	}()
	time.Sleep(10 * time.Millisecond)
	cnl()
	assert.True(t, <-ch)
}

func TestRun_Process_Error_InvalidStrategy(t *testing.T) {
	cnr := mockConsumer{
		chMsg: make(chan Message, 10),
		chErr: make(chan error, 10),
	}
	proc := mockProcessor{retError: true}
	cmp, err := New("test", proc.Process, &mockConsumerFactory{c: &cnr})
	cmp.failStrategy = 4
	assert.NoError(t, err)
	ctx := context.Background()
	cnr.chMsg <- &mockMessage{ctx: ctx}
	err = cmp.Run(ctx)
	assert.Error(t, err)
}

func TestRun_ConsumeError(t *testing.T) {
	cnr := mockConsumer{
		chMsg: make(chan Message, 10),
		chErr: make(chan error, 10),
	}
	proc := mockProcessor{retError: true}
	cmp, err := New("test", proc.Process, &mockConsumerFactory{c: &cnr})
	assert.NoError(t, err)
	ctx := context.Background()
	cnr.chErr <- errors.New("CONSUMER ERROR")
	err = cmp.Run(ctx)
	assert.Error(t, err)
}

func TestRun_ConsumeError_WithRetry(t *testing.T) {
	proc := mockProcessor{retError: true}
	cmp, err := New("test", proc.Process, &mockConsumerFactory{retErr: true}, ConsumerRetry(3, 2*time.Millisecond))
	assert.NoError(t, err)
	ctx := context.Background()
	err = cmp.Run(ctx)
	assert.Error(t, err)
}

func TestRun_Process_Shutdown(t *testing.T) {
	cnr := mockConsumer{
		chMsg: make(chan Message, 10),
		chErr: make(chan error, 10),
	}
	proc := mockProcessor{retError: false}
	cmp, err := New("test", proc.Process, &mockConsumerFactory{c: &cnr})
	assert.NoError(t, err)
	cnr.chMsg <- &mockMessage{ctx: context.Background()}
	ch := make(chan bool)
	ctx, cnl := context.WithCancel(context.Background())
	go func() {
		err1 := cmp.Run(ctx)
		assert.NoError(t, err1)
		ch <- true
	}()
	time.Sleep(10 * time.Millisecond)
	cnl()
	assert.True(t, <-ch)
}

type mockMessage struct {
	ctx context.Context
}

func (mm *mockMessage) Context() context.Context {
	return mm.ctx
}

func (mm *mockMessage) Decode(v interface{}) error {
	return nil
}

func (mm *mockMessage) Ack() error {
	return nil
}

func (mm *mockMessage) Nack() error {
	return nil
}

type mockProcessor struct {
	retError bool
}

func (mp *mockProcessor) Process(msg Message) error {
	if mp.retError {
		return errors.New("PROC ERROR")
	}
	return nil
}

type mockConsumerFactory struct {
	c      Consumer
	retErr bool
}

func (mcf *mockConsumerFactory) Create() (Consumer, error) {
	if mcf.retErr {
		return nil, errors.New("FACTORY ERROR")
	}
	return mcf.c, nil
}

type mockConsumer struct {
	consumeError bool
	chMsg        chan Message
	chErr        chan error
}

func (mc *mockConsumer) SetTimeout(timeout time.Duration) {
}

func (mc *mockConsumer) Consume(context.Context) (<-chan Message, <-chan error, error) {
	if mc.consumeError {
		return nil, nil, errors.New("CONSUMER ERROR")
	}
	return mc.chMsg, mc.chErr, nil
}

func (mc *mockConsumer) Close() error {
	return nil
}
