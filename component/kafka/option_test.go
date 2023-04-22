package kafka

import (
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
)

func TestFailureStrategy(t *testing.T) {
	t.Parallel()
	type args struct {
		strategy FailStrategy
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success-exit": {
			args: args{strategy: ExitStrategy},
		},
		"success-skip": {
			args: args{strategy: SkipStrategy},
		},
		"invalid strategy": {
			args:        args{strategy: -1},
			expectedErr: "invalid failure strategy provided",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := WithFailureStrategy(tt.args.strategy)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.failStrategy, tt.args.strategy)
			}
		})
	}
}

func TestRetries(t *testing.T) {
	c := &Component{}
	err := WithRetries(20)(c)
	assert.NoError(t, err)
	assert.Equal(t, c.retries, uint(20))
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
				assert.Equal(t, c.retryWait, tt.args.retryWait)
			}
		})
	}
}

func TestBatchSize(t *testing.T) {
	t.Parallel()
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
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := WithBatchSize(tt.args.batchSize)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.batchSize, tt.args.batchSize)
			}
		})
	}
}

func TestBatchTimeout(t *testing.T) {
	t.Parallel()
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
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := WithBatchTimeout(tt.args.batchTimeout)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.batchTimeout, tt.args.batchTimeout)
			}
		})
	}
}

func TestNewSessionCallback(t *testing.T) {
	t.Parallel()
	type args struct {
		sessionCallback func(sarama.ConsumerGroupSession) error
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{sessionCallback: func(session sarama.ConsumerGroupSession) error {
				return nil
			}},
		},
		"nil session callback": {
			args:        args{},
			expectedErr: "nil session callback",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := WithNewSessionCallback(tt.args.sessionCallback)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, c.sessionCallback)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, c.sessionCallback)
			}
		})
	}
}

func TestBatchMessageDeduplication(t *testing.T) {
	c := &Component{}
	err := WithBatchMessageDeduplication()(c)
	assert.NoError(t, err)
	assert.Equal(t, c.batchMessageDeduplication, true)
}
