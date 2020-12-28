package sqs

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

func TestMaxMessages(t *testing.T) {
	type args struct {
		maxMessages *int64
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{maxMessages: aws.Int64(5)},
		},
		"zero message size": {
			args:        args{maxMessages: aws.Int64(0)},
			expectedErr: "max messages should be between 1 and 10",
		},
		"over max message size": {
			args:        args{maxMessages: aws.Int64(11)},
			expectedErr: "max messages should be between 1 and 10",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := MaxMessages(*tt.args.maxMessages)(c)
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
	type args struct {
		waitSeconds *int64
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{waitSeconds: aws.Int64(5)},
		},
		"negative message size": {
			args:        args{waitSeconds: aws.Int64(-1)},
			expectedErr: "poll wait seconds should be between 0 and 20",
		},
		"over max wait seconds": {
			args:        args{waitSeconds: aws.Int64(21)},
			expectedErr: "poll wait seconds should be between 0 and 20",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := PollWaitSeconds(*tt.args.waitSeconds)(c)
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
	type args struct {
		timeout *int64
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{timeout: aws.Int64(5)},
		},
		"negative message size": {
			args:        args{timeout: aws.Int64(-1)},
			expectedErr: "visibility timeout should be between 0 and 43200 seconds",
		},
		"over max wait seconds": {
			args:        args{timeout: aws.Int64(twelveHoursInSeconds + 1)},
			expectedErr: "visibility timeout should be between 0 and 43200 seconds",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := VisibilityTimeout(*tt.args.timeout)(c)
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
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := QueueStatsInterval(tt.args.interval)(c)
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
	err := Retries(20)(c)
	assert.NoError(t, err)
	assert.Equal(t, c.retry.count, uint(20))
}

func TestRetryWait(t *testing.T) {
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
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := RetryWait(tt.args.retryWait)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.retry.wait, tt.args.retryWait)
			}
		})
	}
}
