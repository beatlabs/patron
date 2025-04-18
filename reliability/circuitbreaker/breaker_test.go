package circuitbreaker

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	validSetting := Setting{RetrySuccessThreshold: uint(1), MaxRetryExecutionThreshold: 1}
	invalidSetting := Setting{RetrySuccessThreshold: 2, MaxRetryExecutionThreshold: 1}
	type args struct {
		name string
		s    Setting
	}
	tests := map[string]struct {
		args    args
		wantErr bool
	}{
		"success":          {args: args{name: "test", s: validSetting}, wantErr: false},
		"missing name":     {args: args{name: "", s: validSetting}, wantErr: true},
		"invalid settings": {args: args{name: "test", s: invalidSetting}, wantErr: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := New(tt.args.name, tt.args.s)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestCircuitBreaker_isHalfOpen(t *testing.T) {
	t.Parallel()

	type fields struct {
		status    status
		nextRetry int64
	}
	tests := map[string]struct {
		fields fields
		want   bool
	}{
		"closed":    {fields: fields{status: closed, nextRetry: tsFuture}, want: false},
		"open":      {fields: fields{status: opened, nextRetry: time.Now().Add(1 * time.Hour).UnixNano()}, want: false},
		"half open": {fields: fields{status: opened, nextRetry: time.Now().Add(-1 * time.Minute).UnixNano()}, want: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cb := &CircuitBreaker{
				status:    tt.fields.status,
				nextRetry: tt.fields.nextRetry,
			}
			assert.Equal(t, tt.want, cb.isHalfOpen())
		})
	}
}

func TestCircuitBreaker_isOpen(t *testing.T) {
	t.Parallel()

	type fields struct {
		status    status
		nextRetry int64
	}
	tests := map[string]struct {
		fields fields
		want   bool
	}{
		"closed":    {fields: fields{status: closed, nextRetry: tsFuture}, want: false},
		"half open": {fields: fields{status: opened, nextRetry: time.Now().Add(-1 * time.Minute).UnixNano()}, want: false},
		"open":      {fields: fields{status: opened, nextRetry: time.Now().Add(1 * time.Hour).UnixNano()}, want: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cb := &CircuitBreaker{
				status:    tt.fields.status,
				nextRetry: tt.fields.nextRetry,
			}
			assert.Equal(t, tt.want, cb.isOpen())
		})
	}
}

func TestCircuitBreaker_isClose(t *testing.T) {
	t.Parallel()

	type fields struct {
		status    status
		nextRetry int64
	}
	tests := map[string]struct {
		fields fields
		want   bool
	}{
		"closed":    {fields: fields{status: closed, nextRetry: tsFuture}, want: true},
		"half open": {fields: fields{status: opened, nextRetry: time.Now().Add(-1 * time.Minute).UnixNano()}, want: false},
		"open":      {fields: fields{status: opened, nextRetry: time.Now().Add(1 * time.Hour).UnixNano()}, want: false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cb := &CircuitBreaker{
				status:    tt.fields.status,
				nextRetry: tt.fields.nextRetry,
			}
			assert.Equal(t, tt.want, cb.isClose())
		})
	}
}

func TestCircuitBreaker_Close_Open_HalfOpen_Open_HalfOpen_Close(t *testing.T) {
	retryTimeout := 5 * time.Millisecond
	waitRetryTimeout := 7 * time.Millisecond

	set := Setting{FailureThreshold: uint(1), RetryTimeout: retryTimeout, RetrySuccessThreshold: 2, MaxRetryExecutionThreshold: 2}
	cb, err := New("test", set)
	require.NoError(t, err)
	_, err = cb.Execute(testSuccessAction)
	require.NoError(t, err)
	assert.Equal(t, uint(0), cb.failures)
	assert.Equal(t, uint(0), cb.executions)
	assert.Equal(t, uint(0), cb.retries)
	assert.True(t, cb.isClose())
	assert.Equal(t, tsFuture, cb.nextRetry)
	// will transition to open
	_, err = cb.Execute(testFailureAction)
	require.EqualError(t, err, "test error")
	assert.Equal(t, uint(0), cb.failures)
	assert.Equal(t, uint(0), cb.executions)
	assert.Equal(t, uint(0), cb.retries)
	assert.True(t, cb.isOpen())
	assert.Less(t, cb.nextRetry, tsFuture)
	// open, returns err immediately
	_, err = cb.Execute(testSuccessAction)
	require.EqualError(t, err, "circuit is open")
	assert.Equal(t, uint(0), cb.failures)
	assert.Equal(t, uint(0), cb.executions)
	assert.Equal(t, uint(0), cb.retries)
	assert.True(t, cb.isOpen())
	assert.Less(t, cb.nextRetry, tsFuture)
	// should be half open now and will stay in there
	time.Sleep(waitRetryTimeout)
	_, err = cb.Execute(testFailureAction)
	require.EqualError(t, err, "test error")
	assert.Equal(t, uint(1), cb.failures)
	assert.Equal(t, uint(1), cb.executions)
	assert.Equal(t, uint(0), cb.retries)
	assert.True(t, cb.isHalfOpen())
	assert.Less(t, cb.nextRetry, tsFuture)
	// should be half open now and will transition to open
	_, err = cb.Execute(testFailureAction)
	require.EqualError(t, err, "test error")
	assert.Equal(t, uint(0), cb.failures)
	assert.Equal(t, uint(0), cb.executions)
	assert.Equal(t, uint(0), cb.retries)
	assert.True(t, cb.isOpen())
	assert.Less(t, cb.nextRetry, tsFuture)
	// should be half open now and will transition to close
	time.Sleep(waitRetryTimeout)
	_, err = cb.Execute(testSuccessAction)
	require.NoError(t, err)
	assert.Equal(t, uint(0), cb.failures)
	assert.Equal(t, uint(1), cb.executions)
	assert.Equal(t, uint(1), cb.retries)
	assert.True(t, cb.isHalfOpen())
	assert.Less(t, cb.nextRetry, tsFuture)
	_, err = cb.Execute(testSuccessAction)
	require.NoError(t, err)
	assert.Equal(t, uint(0), cb.failures)
	assert.Equal(t, uint(0), cb.executions)
	assert.Equal(t, uint(0), cb.retries)
	assert.True(t, cb.isClose())
	assert.Equal(t, tsFuture, cb.nextRetry)
}

func BenchmarkCircuitBreaker_Execute(b *testing.B) {
	set := Setting{
		FailureThreshold:           uint(1),
		RetryTimeout:               1 * time.Second,
		RetrySuccessThreshold:      uint(1),
		MaxRetryExecutionThreshold: 1,
	}
	cb, _ := New("test", set)

	for b.Loop() {
		cb.Execute(testFailureAction) // nolint:errcheck
	}
}

func testSuccessAction() (any, error) {
	return "test", nil
}

func testFailureAction() (any, error) {
	return "", errors.New("test error")
}
