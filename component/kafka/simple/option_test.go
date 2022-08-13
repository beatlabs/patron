package simple

import (
	"testing"
	"time"

	"github.com/Shopify/sarama"
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
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := FailureStrategy(tt.args.strategy)(c)
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
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := RetryWait(tt.args.retryWait)(c)
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
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := BatchSize(tt.args.batchSize)(c)
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
			expectedErr: "batch timeout should greater than zero",
		},
		"zero batch timeout": {
			args:        args{batchTimeout: 0 * time.Second},
			expectedErr: "batch timeout should greater than zero",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := BatchTimeout(tt.args.batchTimeout)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.batchTimeout, tt.args.batchTimeout)
			}
		})
	}
}

func TestSaramaConfig(t *testing.T) {
	saramaCfg := sarama.NewConfig()
	// batches will be responsible for committing
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = false
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaCfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky
	saramaCfg.Net.DialTimeout = 15 * time.Second
	saramaCfg.Version = sarama.V2_6_0_0
	c := &Component{}
	err := SaramaConfig(saramaCfg)(c)
	assert.NoError(t, err)
	assert.Equal(t, c.saramaConfig, saramaCfg)
}

func TestDurationOffset(t *testing.T) {
	type args struct {
		since         time.Duration
		timeExtractor TimeExtractor
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{
				since: time.Second,
				timeExtractor: func(message *sarama.ConsumerMessage) (time.Time, error) {
					return time.Time{}, nil
				},
			},
		},
		"invalid duration": {
			args: args{
				since: -time.Second,
				timeExtractor: func(message *sarama.ConsumerMessage) (time.Time, error) {
					return time.Time{}, nil
				},
			},
			expectedErr: "duration must be positive",
		},
		"missing time extractor": {
			args: args{
				since: time.Second,
			},
			expectedErr: "empty time extractor function",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := DurationOffset(tt.args.since, tt.args.timeExtractor)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				c.durationBasedConsumer = true
				assert.Equal(t, c.durationOffset, tt.args.since)
				assert.NotNil(t, c.timeExtractor)
			}
		})
	}
}

func TestNotificationOnceReachingLatestOffset(t *testing.T) {
	type args struct {
		ch chan<- struct{}
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{
				ch: make(chan struct{}),
			},
		},
		"missing channel": {
			args:        args{},
			expectedErr: "nil channel provided",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			c := &Component{}
			err := NotificationOnceReachingLatestOffset(tt.args.ch)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
				c.durationBasedConsumer = true
				assert.Equal(t, c.latestOffsetReachedChan, tt.args.ch)
			}
		})
	}
}
