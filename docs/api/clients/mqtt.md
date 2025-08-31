# MQTT client (publisher)

MQTT v5 publisher with OpenTelemetry tracing, correlation propagation, and publish duration metrics.

- Package: `github.com/beatlabs/patron/client/mqtt`
- Upstream: `github.com/eclipse/paho.golang/autopaho` and `github.com/eclipse/paho.golang/paho`

## Usage

```go
cfg, _ := mqtt.DefaultConfig(brokers, clientID)
p, err := mqtt.New(ctx, cfg)

defer p.Disconnect(ctx)

pub := &paho.Publish{Topic: "topic", Payload: []byte("hello")}
_, err = p.Publish(ctx, pub)
```

- Correlation and OTEL headers are injected automatically into user properties.
- Metric `mqtt.publish.duration` includes topic and status.
