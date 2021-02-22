package amqp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
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
				oo:    []OptionFunc{Batching(5, 5*time.Millisecond)},
			},
		},
		"missing url": {
			args: args{
				url:   "",
				queue: "queue",
				proc:  proc,
				oo:    []OptionFunc{Batching(5, 5*time.Millisecond)},
			},
			expectedErr: "url is empty",
		},
		"missing queue": {
			args: args{
				url:   "url",
				queue: "",
				proc:  proc,
				oo:    []OptionFunc{Batching(5, 5*time.Millisecond)},
			},
			expectedErr: "queue is empty",
		},
		"missing process function": {
			args: args{
				url:   "url",
				queue: "queue",
				proc:  nil,
				oo:    []OptionFunc{Batching(5, 5*time.Millisecond)},
			},
			expectedErr: "process function is nil",
		},
		"batching option fails": {
			args: args{
				url:   "url",
				queue: "queue",
				proc:  proc,
				oo:    []OptionFunc{Batching(0, 5*time.Millisecond)},
			},
			expectedErr: "count should be larger than 1 message",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := New(tt.args.url, tt.args.queue, tt.args.proc, tt.args.oo...)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}
