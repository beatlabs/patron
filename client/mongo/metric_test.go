package mongo

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/event"
)

func TestObservabilityMonitor_Started(t *testing.T) {
	t.Parallel()

	callbackCalled := false
	traceMonitor := &event.CommandMonitor{
		Started: func(_ context.Context, _ *event.CommandStartedEvent) {
			callbackCalled = true
		},
	}

	monitor, err := newObservabilityMonitor(traceMonitor)
	require.NoError(t, err)
	ctx := context.Background()

	evt := &event.CommandStartedEvent{
		CommandName:  "find",
		DatabaseName: "test",
	}

	monitor.Started(ctx, evt)

	assert.True(t, callbackCalled, "trace monitor Started callback should be called")
}

func TestObservabilityMonitor_Succeeded(t *testing.T) {
	t.Parallel()

	callbackCalled := false
	traceMonitor := &event.CommandMonitor{
		Succeeded: func(_ context.Context, _ *event.CommandSucceededEvent) {
			callbackCalled = true
		},
	}

	monitor, err := newObservabilityMonitor(traceMonitor)
	require.NoError(t, err)
	ctx := context.Background()

	evt := &event.CommandSucceededEvent{
		CommandFinishedEvent: event.CommandFinishedEvent{
			Duration: 100 * time.Millisecond,
		},
	}

	monitor.Succeeded(ctx, evt)

	assert.True(t, callbackCalled, "trace monitor Succeeded callback should be called")
}

func TestObservabilityMonitor_Failed(t *testing.T) {
	t.Parallel()

	callbackCalled := false
	traceMonitor := &event.CommandMonitor{
		Failed: func(_ context.Context, _ *event.CommandFailedEvent) {
			callbackCalled = true
		},
	}

	monitor, err := newObservabilityMonitor(traceMonitor)
	require.NoError(t, err)
	ctx := context.Background()

	evt := &event.CommandFailedEvent{
		CommandFinishedEvent: event.CommandFinishedEvent{
			Duration: 50 * time.Millisecond,
		},
		Failure: "connection timeout",
	}

	monitor.Failed(ctx, evt)

	assert.True(t, callbackCalled, "trace monitor Failed callback should be called")
}

func TestCommandAttr(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		cmdName  string
		expected string
	}{
		"find command": {
			cmdName:  "find",
			expected: "find",
		},
		"insert command": {
			cmdName:  "insert",
			expected: "insert",
		},
		"update command": {
			cmdName:  "update",
			expected: "update",
		},
		"delete command": {
			cmdName:  "delete",
			expected: "delete",
		},
		"empty command": {
			cmdName:  "",
			expected: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			attr := commandAttr(tt.cmdName)

			require.NotNil(t, attr)
			assert.Equal(t, "command", string(attr.Key))
			assert.Equal(t, tt.expected, attr.Value.AsString())
		})
	}
}

func TestObservabilityMonitor_Integration(t *testing.T) {
	t.Parallel()

	t.Run("all callbacks work together", func(t *testing.T) {
		t.Parallel()

		startedCalled := false
		succeededCalled := false
		failedCalled := false

		traceMonitor := &event.CommandMonitor{
			Started: func(_ context.Context, _ *event.CommandStartedEvent) {
				startedCalled = true
			},
			Succeeded: func(_ context.Context, _ *event.CommandSucceededEvent) {
				succeededCalled = true
			},
			Failed: func(_ context.Context, _ *event.CommandFailedEvent) {
				failedCalled = true
			},
		}

		monitor, err := newObservabilityMonitor(traceMonitor)
		require.NoError(t, err)
		ctx := context.Background()

		// Simulate a command lifecycle
		startEvt := &event.CommandStartedEvent{
			CommandName: "find",
		}
		monitor.Started(ctx, startEvt)

		successEvt := &event.CommandSucceededEvent{
			CommandFinishedEvent: event.CommandFinishedEvent{
				Duration: 10 * time.Millisecond,
			},
		}
		monitor.Succeeded(ctx, successEvt)

		assert.True(t, startedCalled, "Started should be called")
		assert.True(t, succeededCalled, "Succeeded should be called")
		assert.False(t, failedCalled, "Failed should not be called in success case")
	})

	t.Run("failure path", func(t *testing.T) {
		t.Parallel()

		startedCalled := false
		failedCalled := false

		traceMonitor := &event.CommandMonitor{
			Started: func(_ context.Context, _ *event.CommandStartedEvent) {
				startedCalled = true
			},
			Failed: func(_ context.Context, _ *event.CommandFailedEvent) {
				failedCalled = true
			},
		}

		monitor, err := newObservabilityMonitor(traceMonitor)
		require.NoError(t, err)
		ctx := context.Background()

		// Simulate a failed command
		startEvt := &event.CommandStartedEvent{
			CommandName: "find",
		}
		monitor.Started(ctx, startEvt)

		failEvt := &event.CommandFailedEvent{
			CommandFinishedEvent: event.CommandFinishedEvent{
				Duration: 5 * time.Millisecond,
			},
			Failure: "network error",
		}
		monitor.Failed(ctx, failEvt)

		assert.True(t, startedCalled, "Started should be called")
		assert.True(t, failedCalled, "Failed should be called")
	})
}
