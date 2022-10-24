package sqs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMaxMessages(t *testing.T) {
	t.Parallel()
	type args struct {
		maxMessages int32
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
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := WithMaxMessages(tt.args.maxMessages)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.cfg.maxMessages, tt.args.maxMessages)
			}
		})
	}
}

func TestPollWaitSeconds(t *testing.T) {
	t.Parallel()
	type args struct {
		waitSeconds int32
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
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := WithPollWaitSeconds(tt.args.waitSeconds)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.cfg.pollWaitSeconds, tt.args.waitSeconds)
			}
		})
	}
}

func TestVisibilityTimeout(t *testing.T) {
	t.Parallel()
	type args struct {
		timeout int32
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
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := WithVisibilityTimeout(tt.args.timeout)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.cfg.visibilityTimeout, tt.args.timeout)
			}
		})
	}
}

func TestQueueStatsInterval(t *testing.T) {
	t.Parallel()
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
			expectedErr: "sqsAPI stats interval should be a positive value",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := WithQueueStatsInterval(tt.args.interval)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.stats.interval, tt.args.interval)
			}
		})
	}
}

func TestRetries(t *testing.T) {
	c := &Component{}
	err := WithRetries(20)(c)
	assert.NoError(t, err)
	assert.Equal(t, c.retry.count, uint(20))
}

func TestRetryWait(t *testing.T) {
	t.Parallel()
	type args struct {
		retryWait time.Duration
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{retryWait: 5 * time.Second},
		},
		"negative retry wait": {
			args:        args{retryWait: -1 * time.Second},
			expectedErr: "retry wait time should be a positive number",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := WithRetryWait(tt.args.retryWait)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.retry.wait, tt.args.retryWait)
			}
		})
	}
}

func TestQueueOwner(t *testing.T) {
	t.Parallel()
	type args struct {
		queueOwner string
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{queueOwner: "10201020"},
		},
		"empty queue owner": {
			args:        args{queueOwner: ""},
			expectedErr: "queue owner should not be empty",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := WithQueueOwner(tt.args.queueOwner)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.queueOwner, tt.args.queueOwner)
			}
		})
	}
}
