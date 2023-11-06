// Package mqtt provides an instrumented publisher for MQTT v5.
package mqtt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"time"

	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/trace"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus"
)

const componentType = "mqtt-publisher"

var publishDurationMetrics *prometheus.HistogramVec

func init() {
	publishDurationMetrics = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "client",
			Subsystem: "mqtt",
			Name:      "publish_duration_seconds",
			Help:      "MQTT publish completed by the client.",
		},
		[]string{"topic", "success"},
	)
	prometheus.MustRegister(publishDurationMetrics)
}

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
			slog.Error("failed to connect", slog.Any("error", err))
		},
		ClientConfig: paho.ClientConfig{
			ClientID: clientID,
			OnServerDisconnect: func(disconnect *paho.Disconnect) {
				slog.Warn("server disconnect received", slog.Int64("reason", int64(disconnect.ReasonCode)))
			},
			OnClientError: func(err error) {
				slog.Error("client failure", slog.Any("error", err))
			},
			PublishHook: func(publish *paho.Publish) {
				slog.Debug("message published", slog.String("topic", publish.Topic))
			},
		},
	}, nil
}

// Publisher definition.
type Publisher struct {
	cm *autopaho.ConnectionManager
}

// New creates a publisher.
func New(ctx context.Context, cfg autopaho.ClientConfig) (*Publisher, error) {
	cm, err := autopaho.NewConnection(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection manager: %w", err)
	}

	return &Publisher{cm: cm}, nil
}

// Publish provides a instrumented publishing of a message.
func (p *Publisher) Publish(ctx context.Context, pub *paho.Publish) (*paho.PublishResponse, error) {
	sp, _ := trace.ChildSpan(ctx, trace.ComponentOpName(componentType, pub.Topic), componentType,
		ext.SpanKindProducer, opentracing.Tag{Key: "topic", Value: pub.Topic})

	start := time.Now()

	err := p.cm.AwaitConnection(ctx)
	if err != nil {
		observePublish(ctx, sp, start, pub.Topic, err)
		return nil, fmt.Errorf("connection is not up: %w", err)
	}

	if err = injectObservabilityHeaders(ctx, pub, sp); err != nil {
		observePublish(ctx, sp, start, pub.Topic, err)
		return nil, fmt.Errorf("failed to inject tracing headers: %w", err)
	}

	rsp, err := p.cm.Publish(ctx, pub)
	if err != nil {
		observePublish(ctx, sp, start, pub.Topic, err)
		return nil, fmt.Errorf("failed to publish message: %w", err)
	}

	observePublish(ctx, sp, start, pub.Topic, err)
	return rsp, nil
}

// Disconnect from the broker.
func (p *Publisher) Disconnect(ctx context.Context) error {
	return p.cm.Disconnect(ctx)
}

type mqttHeadersCarrier paho.UserProperties

// Set implements Set() of opentracing.TextMapWriter.
func (m *mqttHeadersCarrier) Set(key, val string) {
	hdr := paho.UserProperties(*m)
	hdr.Add(key, val)
	*m = mqttHeadersCarrier(hdr)
}

func injectObservabilityHeaders(ctx context.Context, pub *paho.Publish, sp opentracing.Span) error {
	ensurePublishingProperties(pub)
	pub.Properties.User.Add(correlation.HeaderID, correlation.IDFromContext(ctx))

	c := mqttHeadersCarrier(pub.Properties.User)
	err := sp.Tracer().Inject(sp.Context(), opentracing.TextMap, &c)
	pub.Properties.User = paho.UserProperties(c)
	return err
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

func observePublish(ctx context.Context, span opentracing.Span, start time.Time, topic string, err error) {
	trace.SpanComplete(span, err)

	durationHistogram := trace.Histogram{
		Observer: publishDurationMetrics.WithLabelValues(topic, strconv.FormatBool(err == nil)),
	}
	durationHistogram.Observe(ctx, time.Since(start).Seconds())
}
