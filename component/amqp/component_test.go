package amqp

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	proc := func(_ context.Context, b Batch) {
		_, _ = b.ACK()
	}

	type args struct {
		url   string
		queue string
		proc  ProcessorFunc
		oo    []OptionFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{
				url:   "url",
				queue: "queue",
				proc:  proc,
				oo:    []OptionFunc{WithBatching(5, 5*time.Millisecond)},
			},
		},
		"missing url": {
			args: args{
				url:   "",
				queue: "queue",
				proc:  proc,
				oo:    []OptionFunc{WithBatching(5, 5*time.Millisecond)},
			},
			expectedErr: "url is empty",
		},
		"missing queue": {
			args: args{
				url:   "url",
				queue: "",
				proc:  proc,
				oo:    []OptionFunc{WithBatching(5, 5*time.Millisecond)},
			},
			expectedErr: "queue is empty",
		},
		"missing process function": {
			args: args{
				url:   "url",
				queue: "queue",
				proc:  nil,
				oo:    []OptionFunc{WithBatching(5, 5*time.Millisecond)},
			},
			expectedErr: "process function is nil",
		},
		"batching option fails": {
			args: args{
				url:   "url",
				queue: "queue",
				proc:  proc,
				oo:    []OptionFunc{WithBatching(0, 5*time.Millisecond)},
			},
			expectedErr: "count should be larger than 1 message",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := New(tt.args.url, tt.args.queue, tt.args.proc, tt.args.oo...)

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}


func TestLogStatsError(t *testing.T) {
	errOutput := captureAMQPLogOutput(t, func() {
		logStatsError(errors.New("stats failed"))
	})

	assert.Contains(t, errOutput, "failed to report sqsAPI stats")
	assert.NotContains(t, errOutput, "%v")
	assert.Contains(t, errOutput, "stats failed")
}

func captureAMQPLogOutput(t *testing.T, fn func()) string {
	t.Helper()

	originalLogger := slog.Default()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	defer func() {
		slog.SetDefault(originalLogger)
	}()

	slog.SetDefault(logger)
	fn()

	return buf.String()
}
