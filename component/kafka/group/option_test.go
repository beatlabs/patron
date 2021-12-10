package group

import (
	"testing"
	"time"

	"github.com/beatlabs/patron/component/kafka"
	"github.com/stretchr/testify/assert"
)

func TestFailureStrategy(t *testing.T) {
	type args struct {
		strategy kafka.FailStrategy
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success-exit": {
			args: args{strategy: kafka.ExitStrategy},
		},
		"success-skip": {
			args: args{strategy: kafka.SkipStrategy},
		},
		"invalid strategy": {
			args:        args{strategy: -1},
			expectedErr: "invalid failure strategy provided",
		},
	}
	for name, tt := range tests {
		tst := tt
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := FailureStrategy(tst.args.strategy)(c)
			if tst.expectedErr != "" {
				assert.EqualError(t, err, tst.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.failStrategy, tst.args.strategy)
			}
		})
	}
}

func TestRetries(t *testing.T) {
	c := &Component{}
	err := Retries(20)(c)
	assert.NoError(t, err)
	assert.Equal(t, c.retries, uint(20))
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
		tst := tt
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := RetryWait(tst.args.retryWait)(c)
			if tst.expectedErr != "" {
				assert.EqualError(t, err, tst.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.retryWait, tst.args.retryWait)
			}
		})
	}
}

func TestBatchSize(t *testing.T) {
	type args struct {
		batchSize uint
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{batchSize: 1},
		},
		"zero batch size": {
			args:        args{batchSize: 0},
			expectedErr: "zero batch size provided",
		},
	}
	for name, tt := range tests {
		tst := tt
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := BatchSize(tst.args.batchSize)(c)
			if tst.expectedErr != "" {
				assert.EqualError(t, err, tst.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.batchSize, tst.args.batchSize)
			}
		})
	}
}

func TestBatchTimeout(t *testing.T) {
	type args struct {
		batchTimeout time.Duration
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{batchTimeout: 5 * time.Second},
		},
		"negative batch timeout": {
			args:        args{batchTimeout: -1 * time.Second},
			expectedErr: "batch timeout should greater than or equal to zero",
		},
	}
	for name, tt := range tests {
		tst := tt
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := BatchTimeout(tst.args.batchTimeout)(c)
			if tst.expectedErr != "" {
				assert.EqualError(t, err, tst.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.batchTimeout, tst.args.batchTimeout)
			}
		})
	}
}
