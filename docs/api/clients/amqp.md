# AMQP client (RabbitMQ publisher)

A minimal publisher with OpenTelemetry tracing/metrics.

- Package: `github.com/beatlabs/patron/client/amqp`
- Type: `Publisher`
- Constructor: `New(url string, oo ...OptionFunc) (*Publisher, error)`
- Publish: `Publish(ctx, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error`

## Quick start

```go
import (
  patronamqp "github.com/beatlabs/patron/client/amqp"
  amqp "github.com/rabbitmq/amqp091-go"
)

pub, err := patronamqp.New("amqp://guest:guest@localhost:5672/")
if err != nil { /* handle */ }

defer pub.Close()

msg := amqp.Publishing{ContentType: "application/octet-stream", Body: []byte("hello")}
err = pub.Publish(ctx, "exchange-name", "routing-key", false, false, msg)
```

See `examples/client/main.go` (use `-modes=amqp`).

## Options

```go
// Provide custom dial configuration (e.g., timeouts, TLS)
patronamqp.WithConfig(amqp.Config{ /* ... */ })
```

## Observability

- `Publish` starts a producer span, injects trace and correlation headers into AMQP properties, records duration metrics, and sets error status on failure.

## Errors

- Constructor returns error if URL is empty or connection/channel cannot be established.
- `Publish` returns error on broker publish failure.
