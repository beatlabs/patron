package sqs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMaxMessages(t *testing.T) {
	type args struct {
		maxMessages int64
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{maxMessages: 5},
		},
		"zero message size": {
			args:        args{maxMessages: 0},
			expectedErr: "max messages should be between 1 and 10",
		},
		"over max message size": {
			args:        args{maxMessages: 11},
			expectedErr: "max messages should be between 1 and 10",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := NewFactory(&stubQueue{}, "queue")
			require.NoError(t, err)
			err = MaxMessages(tt.args.maxMessages)(f)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, f.maxMessages, tt.args.maxMessages)
			}
		})
	}
}

func TestPollWaitSeconds(t *testing.T) {
	type args struct {
		waitSeconds int64
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{waitSeconds: 5},
		},
		"negative message size": {
			args:        args{waitSeconds: -1},
			expectedErr: "poll wait seconds should be between 0 and 20",
		},
		"over max wait seconds": {
			args:        args{waitSeconds: 21},
			expectedErr: "poll wait seconds should be between 0 and 20",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := NewFactory(&stubQueue{}, "queue")
			require.NoError(t, err)
			err = PollWaitSeconds(tt.args.waitSeconds)(f)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, f.pollWaitSeconds, tt.args.waitSeconds)
			}
		})
	}
}

func TestVisibilityTimeout(t *testing.T) {
	type args struct {
		timeout int64
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{timeout: 5},
		},
		"negative message size": {
			args:        args{timeout: -1},
			expectedErr: "visibility timeout should be between 0 and 43200 seconds",
		},
		"over max wait seconds": {
			args:        args{timeout: twelveHoursInSeconds + 1},
			expectedErr: "visibility timeout should be between 0 and 43200 seconds",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := NewFactory(&stubQueue{}, "queue")
			require.NoError(t, err)
			err = VisibilityTimeout(tt.args.timeout)(f)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, f.visibilityTimeout, tt.args.timeout)
			}
		})
	}
}

func TestBuffer(t *testing.T) {
	type args struct {
		buffer int
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{buffer: 5},
		},
		"negative message size": {
			args:        args{buffer: -1},
			expectedErr: "buffer should be greater or equal to zero",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := NewFactory(&stubQueue{}, "queue")
			require.NoError(t, err)
			err = Buffer(tt.args.buffer)(f)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, f.buffer, tt.args.buffer)
			}
		})
	}
}

func TestQueueStatsInterval(t *testing.T) {
	type args struct {
		interval time.Duration
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{interval: 5 * time.Second},
		},
		"zero interval duration": {
			args:        args{interval: 0},
			expectedErr: "queue stats interval should be a positive value",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f, err := NewFactory(&stubQueue{}, "queue")
			require.NoError(t, err)
			err = QueueStatsInterval(tt.args.interval)(f)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, f.statsInterval, tt.args.interval)
			}
		})
	}
}
