package amqp

import (
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAMQPConfig(t *testing.T) {
	cfg := amqp.Config{Locale: "123"}
	c := &Component{}
	require.NoError(t, WithConfig(cfg)(c))
	assert.Equal(t, cfg, c.cfg)
}

func TestBatching(t *testing.T) {
	t.Parallel()
	type args struct {
		count   uint
		timeout time.Duration
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":         {args: args{count: 2, timeout: 2 * time.Millisecond}},
		"invalid count":   {args: args{count: 1, timeout: 2 * time.Millisecond}, expectedErr: "count should be larger than 1 message"},
		"invalid timeout": {args: args{count: 2, timeout: -3}, expectedErr: "timeout should be a positive number"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := WithBatching(tt.args.count, tt.args.timeout)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, c.batchCfg.count, tt.args.count)
				assert.Equal(t, c.batchCfg.timeout, tt.args.timeout)
			}
		})
	}
}

func TestRetry(t *testing.T) {
	retryCount := uint(5)
	retryDelay := 2 * time.Second
	c := &Component{}
	require.NoError(t, WithRetry(retryCount, retryDelay)(c))
	assert.Equal(t, retryCount, c.retryCfg.count)
	assert.Equal(t, retryDelay, c.retryCfg.delay)
}

func TestRequeue(t *testing.T) {
	c := &Component{}
	require.NoError(t, WithRequeue(false)(c))
	assert.False(t, c.queueCfg.requeue)
}

func TestStatsInterval(t *testing.T) {
	t.Parallel()
	type args struct {
		interval time.Duration
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":          {args: args{interval: 2 * time.Millisecond}},
		"invalid interval": {args: args{interval: -3}, expectedErr: "stats interval should be a positive number"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &Component{}
			err := WithStatsInterval(tt.args.interval)(c)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, c.statsCfg.interval, tt.args.interval)
			}
		})
	}
}
