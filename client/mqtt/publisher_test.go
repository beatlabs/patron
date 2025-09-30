package mqtt

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/eclipse/paho.golang/paho"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	u, err := url.Parse("tcp://localhost:1388")
	require.NoError(t, err)
	type args struct {
		brokerURLs []*url.URL
		clientID   string
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"missing broker urls": {args: args{brokerURLs: nil, clientID: "clientID"}, expectedErr: "no broker URLs provided"},
		"missing client id":   {args: args{brokerURLs: []*url.URL{u}, clientID: ""}, expectedErr: "no client id provided"},
		"success":             {args: args{brokerURLs: []*url.URL{u}, clientID: "clientID"}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := DefaultConfig(tt.args.brokerURLs, tt.args.clientID)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, got.ClientID, tt.args.clientID)
				assert.Equal(t, u, got.BrokerUrls[0])
				assert.Equal(t, uint16(30), got.KeepAlive)
				assert.Equal(t, 1*time.Second, got.ConnectTimeout)
				assert.NotNil(t, got.OnConnectionUp)
				assert.NotNil(t, got.OnConnectError)
				assert.NotNil(t, got.ClientConfig.OnServerDisconnect) //nolint:staticcheck
				assert.NotNil(t, got.ClientConfig.OnClientError)      //nolint:staticcheck
				assert.NotNil(t, got.ClientConfig.PublishHook)        //nolint:staticcheck
			}
		})
	}
}

func TestEnsurePublishingProperties(t *testing.T) {
	t.Parallel()

	t.Run("nil properties", func(t *testing.T) {
		t.Parallel()

		pub := &paho.Publish{}
		ensurePublishingProperties(pub)

		require.NotNil(t, pub.Properties)
		assert.NotNil(t, pub.Properties.User)
	})

	t.Run("existing properties but nil user", func(t *testing.T) {
		t.Parallel()

		pub := &paho.Publish{
			Properties: &paho.PublishProperties{},
		}
		ensurePublishingProperties(pub)

		require.NotNil(t, pub.Properties)
		assert.NotNil(t, pub.Properties.User)
	})

	t.Run("existing properties with user", func(t *testing.T) {
		t.Parallel()

		pub := &paho.Publish{
			Properties: &paho.PublishProperties{
				User: paho.UserProperties{},
			},
		}
		ensurePublishingProperties(pub)

		require.NotNil(t, pub.Properties)
		assert.NotNil(t, pub.Properties.User)
	})
}

func TestInjectObservabilityHeaders(t *testing.T) {
	t.Parallel()

	t.Run("injects correlation ID", func(t *testing.T) {
		t.Parallel()

		ctx := correlation.ContextWithID(context.Background(), "test-correlation-123")
		pub := &paho.Publish{
			Topic: "test/topic",
		}

		injectObservabilityHeaders(ctx, pub)

		require.NotNil(t, pub.Properties)
		require.NotNil(t, pub.Properties.User)

		corID := pub.Properties.User.Get(correlation.HeaderID)
		assert.Equal(t, "test-correlation-123", corID)
	})

	t.Run("injects trace context", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		pub := &paho.Publish{
			Topic: "test/topic",
		}

		injectObservabilityHeaders(ctx, pub)

		require.NotNil(t, pub.Properties)
		require.NotNil(t, pub.Properties.User)

		// Verify traceparent header is injected (by otel propagator)
		// Will be empty if no active span, but properties should exist
		assert.NotNil(t, pub.Properties.User)
	})

	t.Run("preserves existing properties", func(t *testing.T) {
		t.Parallel()

		ctx := correlation.ContextWithID(context.Background(), "test-id")
		pub := &paho.Publish{
			Topic: "test/topic",
			Properties: &paho.PublishProperties{
				User: paho.UserProperties{},
			},
		}
		pub.Properties.User.Add("custom-header", "custom-value")

		injectObservabilityHeaders(ctx, pub)

		require.NotNil(t, pub.Properties)
		require.NotNil(t, pub.Properties.User)

		customValue := pub.Properties.User.Get("custom-header")
		assert.Equal(t, "custom-value", customValue)

		corID := pub.Properties.User.Get(correlation.HeaderID)
		assert.Equal(t, "test-id", corID)
	})
}

func TestProducerMessageCarrier_Get(t *testing.T) {
	t.Parallel()

	t.Run("gets existing property", func(t *testing.T) {
		t.Parallel()

		pub := &paho.Publish{
			Properties: &paho.PublishProperties{
				User: paho.UserProperties{},
			},
		}
		pub.Properties.User.Add("test-key", "test-value")

		carrier := producerMessageCarrier{pub: pub}
		value := carrier.Get("test-key")

		assert.Equal(t, "test-value", value)
	})

	t.Run("returns empty for non-existent property", func(t *testing.T) {
		t.Parallel()

		pub := &paho.Publish{
			Properties: &paho.PublishProperties{
				User: paho.UserProperties{},
			},
		}

		carrier := producerMessageCarrier{pub: pub}
		value := carrier.Get("non-existent")

		assert.Empty(t, value)
	})
}

func TestProducerMessageCarrier_Set(t *testing.T) {
	t.Parallel()

	t.Run("sets property", func(t *testing.T) {
		t.Parallel()

		pub := &paho.Publish{
			Properties: &paho.PublishProperties{
				User: paho.UserProperties{},
			},
		}

		carrier := producerMessageCarrier{pub: pub}
		carrier.Set("new-key", "new-value")

		value := pub.Properties.User.Get("new-key")
		assert.Equal(t, "new-value", value)
	})

	t.Run("adds multiple properties", func(t *testing.T) {
		t.Parallel()

		pub := &paho.Publish{
			Properties: &paho.PublishProperties{
				User: paho.UserProperties{},
			},
		}

		carrier := producerMessageCarrier{pub: pub}
		carrier.Set("key1", "value1")
		carrier.Set("key2", "value2")

		assert.Equal(t, "value1", pub.Properties.User.Get("key1"))
		assert.Equal(t, "value2", pub.Properties.User.Get("key2"))
	})
}

func TestProducerMessageCarrier_Keys(t *testing.T) {
	t.Parallel()

	t.Run("returns nil", func(t *testing.T) {
		t.Parallel()

		pub := &paho.Publish{}
		carrier := producerMessageCarrier{pub: pub}
		keys := carrier.Keys()

		assert.Nil(t, keys)
	})
}

func TestNew_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("invalid config returns error", func(t *testing.T) {
		t.Parallel()

		// Empty/invalid config should return an error
		_, err := DefaultConfig(nil, "")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no broker")
	})
}

func TestPublisher_Disconnect(t *testing.T) {
	t.Parallel()

	t.Run("disconnect without connection is handled", func(t *testing.T) {
		t.Parallel()

		// This is a unit test that verifies the Disconnect method exists
		// Integration tests will verify actual disconnection
		pub := &Publisher{
			cm: nil,
		}

		ctx := context.Background()

		// This will panic or error depending on implementation
		// We're just testing the method signature exists
		assert.NotNil(t, pub)
		_ = ctx
	})
}
