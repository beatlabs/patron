package async

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFailStrategy_String(t *testing.T) {
	tests := []struct {
		name string
		fs   FailStrategy
		want string
	}{
		{name: "NackExitStrategy", fs: NackExitStrategy, want: "NackExitStrategy"},
		{name: "NackStrategy", fs: NackStrategy, want: "NackStrategy"},
		{name: "AckStrategy", fs: AckStrategy, want: "AckStrategy"},
		{name: "Not mapped", fs: -1, want: "N/A"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.fs.String())
		})
	}
}

func TestFailureStrategy(t *testing.T) {
	proc := mockProcessor{}
	type args struct {
		fs FailStrategy
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "success", args: args{fs: NackExitStrategy}, wantErr: false},
		{name: "invalid strategy (lower)", args: args{fs: -1}, wantErr: true},
		{name: "invalid strategy (higher)", args: args{fs: 3}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New("test", proc.Process, &mockConsumerFactory{})
			assert.NoError(t, err)
			err = FailureStrategy(tt.args.fs)(c)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConsumerRetry(t *testing.T) {
	proc := mockProcessor{}
	type args struct {
		retries   int
		retryWait time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "success", args: args{retries: 3, retryWait: time.Second}, wantErr: false},
		{name: "invalid retries", args: args{retries: -1, retryWait: time.Second}, wantErr: true},
		{name: "invalid retry wait", args: args{retries: 3, retryWait: -1}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New("test", proc.Process, &mockConsumerFactory{})
			assert.NoError(t, err)
			err = ConsumerRetry(tt.args.retries, tt.args.retryWait)(c)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestReadinessHandler(t *testing.T) {
	proc := mockProcessor{}
	c, err := New("test", proc.Process, &mockConsumerFactory{})
	assert.NoError(t, err)
	err = ReadinessHandler(func() {})(c)
	assert.NoError(t, err)
}
