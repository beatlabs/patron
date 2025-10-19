// Package mqtt provides an instrumented publisher for MQTT v5.
package mqtt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/observability"
	"github.com/beatlabs/patron/observability/log"
	patronmetric "github.com/beatlabs/patron/observability/metric"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// DefaultConfig provides a config with sane default and logging enabled on the callbacks.
func DefaultConfig(brokerURLs []*url.URL, clientID string) (autopaho.ClientConfig, error) {
	if len(brokerURLs) == 0 {
		return autopaho.ClientConfig{}, errors.New("no broker URLs provided")
	}
	if clientID == "" {
		return autopaho.ClientConfig{}, errors.New("no client id provided")
	}

	return autopaho.ClientConfig{
		BrokerUrls:        brokerURLs,
		KeepAlive:         30,
		ConnectRetryDelay: 5 * time.Second,
		ConnectTimeout:    1 * time.Second,
		OnConnectionUp: func(_ *autopaho.ConnectionManager, conAck *paho.Connack) {
			slog.Info("connection is up", slog.Int64("reason", int64(conAck.ReasonCode)))
		},
		OnConnectError: func(err error) {
			slog.Error("failed to connect", log.ErrorAttr(err))
		},
		ClientConfig: paho.ClientConfig{
			ClientID: clientID,
			OnServerDisconnect: func(disconnect *paho.Disconnect) {
				slog.Warn("server disconnect received", slog.Int64("reason", int64(disconnect.ReasonCode)))
			},
			OnClientError: func(err error) {
				slog.Error("client failure", log.ErrorAttr(err))
			},
			PublishHook: func(publish *paho.Publish) {
				slog.Debug("message published", slog.String("topic", publish.Topic))
			},
		},
	}, nil
}

// Publisher definition.
type Publisher struct {
	cm                *autopaho.ConnectionManager
	durationHistogram metric.Int64Histogram
}

// New creates a publisher.
func New(ctx context.Context, cfg autopaho.ClientConfig) (*Publisher, error) {
	cm, err := autopaho.NewConnection(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	durationHistogram, err := patronmetric.Int64Histogram("mqtt", "mqtt.publish.duration", "MQTT publish duration.", "ms")
	if err != nil {
		return nil, err
	}

	return &Publisher{cm: cm, durationHistogram: durationHistogram}, nil
}

// Publish provides a instrumented publishing of a message.
func (p *Publisher) Publish(ctx context.Context, pub *paho.Publish) (*paho.PublishResponse, error) {
	ctx, sp := patrontrace.StartSpan(ctx, "publish", trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(attribute.String("topic", pub.Topic), observability.ClientAttribute("mqtt")),
	)
	defer sp.End()

	start := time.Now()

	err := p.cm.AwaitConnection(ctx)
	if err != nil {
		p.observePublish(ctx, start, pub.Topic, err)
		return nil, fmt.Errorf("connection is not up: %w", err)
	}

	injectObservabilityHeaders(ctx, pub)

	rsp, err := p.cm.Publish(ctx, pub)
	if err != nil {
		p.observePublish(ctx, start, pub.Topic, err)
		return nil, fmt.Errorf("failed to publish message: %w", err)
	}

	p.observePublish(ctx, start, pub.Topic, nil)
	return rsp, nil
}

// Disconnect from the broker.
func (p *Publisher) Disconnect(ctx context.Context) error {
	return p.cm.Disconnect(ctx)
}

func injectObservabilityHeaders(ctx context.Context, pub *paho.Publish) {
	ensurePublishingProperties(pub)

	ensurePublishingProperties(pub)
	otel.GetTextMapPropagator().Inject(ctx, producerMessageCarrier{pub})

	pub.Properties.User.Add(correlation.HeaderID, correlation.IDFromContext(ctx))
}

func ensurePublishingProperties(pub *paho.Publish) {
	if pub.Properties == nil {
		pub.Properties = &paho.PublishProperties{
			User: paho.UserProperties{},
		}
		return
	}
	if pub.Properties.User == nil {
		pub.Properties.User = paho.UserProperties{}
	}
}

type producerMessageCarrier struct {
	pub *paho.Publish
}

// Get retrieves a single value for a given key.
func (c producerMessageCarrier) Get(key string) string {
	return c.pub.Properties.User.Get(key)
}

// Set sets a header.
func (c producerMessageCarrier) Set(key, val string) {
	c.pub.Properties.User.Add(key, val)
}

// Keys returns a slice of all key identifiers in the carrier.
func (c producerMessageCarrier) Keys() []string {
	return nil
}
