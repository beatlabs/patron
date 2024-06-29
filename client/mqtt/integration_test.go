//go:build integration

package mqtt

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/beatlabs/patron/observability/trace"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const (
	testTopic = "testTopic"
	hiveMQURL = "tcp://localhost:1883"
)

func TestPublish(t *testing.T) {
	// Trace monitoring setup
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	// Metrics monitoring setup
	read := metricsdk.NewManualReader()
	provider := metricsdk.NewMeterProvider(metricsdk.WithReader(read))
	defer func() {
		assert.NoError(t, provider.Shutdown(context.Background()))
	}()

	otel.SetMeterProvider(provider)

	u, err := url.Parse(hiveMQURL)
	require.NoError(t, err)

	var gotPub *paho.Publish
	chDone := make(chan struct{})

	router := paho.NewStandardRouterWithDefault(func(m *paho.Publish) {
		gotPub = m
		chDone <- struct{}{}
	})

	cmSub, err := createSubscriber(t, u, router)
	require.NoError(t, err)

	cfg, err := DefaultConfig([]*url.URL{u}, "test-publisher")
	require.NoError(t, err)

	ctx, cnl := context.WithCancel(context.Background())
	defer cnl()

	pub, err := New(ctx, cfg)
	require.NoError(t, err)

	payload, err := json.Marshal(struct{ Count uint64 }{Count: 1})
	require.NoError(t, err)

	msg := &paho.Publish{
		QoS:     1,
		Topic:   testTopic,
		Payload: payload,
	}

	rsp, err := pub.Publish(ctx, msg)
	require.NoError(t, err)
	assert.NotNil(t, rsp)

	require.NoError(t, pub.Disconnect(ctx))

	// Traces
	assert.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expected := tracetest.SpanStub{
		Name: "publish",
		Attributes: []attribute.KeyValue{
			attribute.String("topic", testTopic),
			attribute.String("client", "mqtt"),
		},
	}

	snaps := exp.GetSpans().Snapshots()

	assert.Len(t, snaps, 1)
	assert.Equal(t, expected.Name, snaps[0].Name())
	assert.Equal(t, expected.Attributes, snaps[0].Attributes())

	// Metrics
	collectedMetrics := &metricdata.ResourceMetrics{}
	assert.NoError(t, read.Collect(context.Background(), collectedMetrics))
	assert.Equal(t, 1, len(collectedMetrics.ScopeMetrics))

	<-chDone
	require.NoError(t, cmSub.Disconnect(context.Background()))

	msg.PacketID = gotPub.PacketID
	assert.Equal(t, msg, gotPub)
}

func createSubscriber(t *testing.T, u *url.URL, router paho.Router) (*autopaho.ConnectionManager, error) {
	cfg := autopaho.ClientConfig{
		BrokerUrls:        []*url.URL{u},
		KeepAlive:         30,
		ConnectRetryDelay: 5 * time.Second,
		ConnectTimeout:    1 * time.Second,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, _ *paho.Connack) {
			_, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{
						Topic: testTopic,
						QoS:   1,
					},
				},
			})
			require.NoError(t, err)
		},
		ClientConfig: paho.ClientConfig{
			ClientID: "test-subscriber",
			Router:   router,
		},
	}

	cm, err := autopaho.NewConnection(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	err = cm.AwaitConnection(context.Background())
	if err != nil {
		return nil, err
	}

	return cm, nil
}
