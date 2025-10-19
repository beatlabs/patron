package mongo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestConnect_ValidationErrors(t *testing.T) {
	t.Parallel()

	t.Run("invalid URI returns error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		opts := options.Client().ApplyURI("invalid://uri")

		client, err := Connect(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, client)
	})

	t.Run("empty URI returns error", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		opts := options.Client().ApplyURI("")

		client, err := Connect(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestConnect_WithMonitor(t *testing.T) {
	t.Parallel()

	t.Run("monitor is set up", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		// Use a valid URI format but point to a non-existent host
		// This will allow us to test that Connect is called with the monitor
		opts := options.Client().ApplyURI("mongodb://localhost:27017")

		client, err := Connect(ctx, opts)

		// We expect no error on Connect (connection is lazy), but we should have a client
		require.NoError(t, err)
		require.NotNil(t, client)

		// Verify the client was created
		err = client.Disconnect(ctx)
		require.NoError(t, err)
	})

	t.Run("multiple options are merged", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		opts1 := options.Client().ApplyURI("mongodb://localhost:27017")
		opts2 := options.Client().SetAppName("test-app")
		opts3 := options.Client().SetMaxPoolSize(10)

		client, err := Connect(ctx, opts1, opts2, opts3)

		require.NoError(t, err)
		require.NotNil(t, client)

		// Clean up
		err = client.Disconnect(ctx)
		require.NoError(t, err)
	})
}

func TestConnect_WithContext(t *testing.T) {
	t.Parallel()

	t.Run("respects context cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		opts := options.Client().ApplyURI("mongodb://localhost:27017")

		client, err := Connect(ctx, opts)

		// Should succeed even with cancelled context (connection is lazy)
		// but operations would fail
		require.NoError(t, err)
		require.NotNil(t, client)

		// Clean up
		err = client.Disconnect(context.Background())
		require.NoError(t, err)
	})
}

func TestNewObservabilityMonitor(t *testing.T) {
	t.Parallel()

	t.Run("creates monitor with callbacks", func(t *testing.T) {
		t.Parallel()

		traceMonitor := &event.CommandMonitor{
			Started: func(_ context.Context, _ *event.CommandStartedEvent) {
				// Test callback
			},
			Succeeded: func(_ context.Context, _ *event.CommandSucceededEvent) {
				// Test callback
			},
			Failed: func(_ context.Context, _ *event.CommandFailedEvent) {
				// Test callback
			},
		}

		monitor, err := newObservabilityMonitor(traceMonitor)
		require.NoError(t, err)

		require.NotNil(t, monitor)
		assert.NotNil(t, monitor.Started)
		assert.NotNil(t, monitor.Succeeded)
		assert.NotNil(t, monitor.Failed)
	})
}
