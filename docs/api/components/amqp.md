# AMQP component (RabbitMQ consumer)

Consumes messages from RabbitMQ/AMQP with batching, retries, observability, and correlation out of the box.

- Package: `github.com/beatlabs/patron/component/amqp`
- Type: component implementing `Run(ctx context.Context) error`
- Built-ins: OpenTelemetry tracing/metrics, correlation ID propagation, periodic queue stats

## Quick start

```go
import (
  patronamqp "github.com/beatlabs/patron/component/amqp"
)

proc := func(ctx context.Context, batch patronamqp.Batch) {
  for _, m := range batch.Messages() {
    // Process m.Body() or m.Message() as needed
    _ = m.ACK() // or m.NACK()
  }
}

cmp, err := patronamqp.New("amqp://guest:guest@localhost:5672/", "queue-name", proc,
  patronamqp.WithRetry(10, time.Second),
)
// add cmp to the service and Run
```

See the runnable example: `examples/service/amqp.go`.

## Options

```go
// Batching: deliver messages to your processor in groups
patronamqp.WithBatching(count /* >1 */, timeout /* >0 */)

// Retries: reconnect on failures
patronamqp.WithRetry(count, delay)

// Custom AMQP dial config: timeouts, TLS, etc
patronamqp.WithConfig(amqp.Config{ /* ... */ })

// Stats interval: queue depth gauge cadence
patronamqp.WithStatsInterval(interval)

// Requeue policy: control NACK requeue behavior
patronamqp.WithRequeue(true|false)
```

Notes

- Batching: When either the count is reached or the timeout elapses, the batch is delivered. Use `Batch.ACK()`/`Batch.NACK()` to handle many at once; it returns any failed messages and a joined error.
- Each message exposes `Context()` with logger/tracing and `Span()`. Call `ACK()` or `NACK()` to end the span; success/error is recorded accordingly.
- The component auto-reconnects with `WithRetry`; after the configured attempts it returns the last error.
- Queue size is periodically observed via passive declare and exported as a metric.

## Message API

- `Message.ID() string`
- `Message.Body() []byte`
- `Message.Message() amqp.Delivery` (raw access)
- `Message.Context() context.Context` (contains correlation ID and request-scoped logger)
- `Message.Span() trace.Span`
- `Message.ACK() error` / `Message.NACK() error`

## Troubleshooting

- Connection issues: check dial `amqp.Config` (heartbeat, TLS, timeouts). You can inject your own Dialer via `Config.Dial`.
- Consumer tag and re-delivery: Patron sets a unique tag per subscription and uses manual acks. Use `WithRequeue(false)` to avoid requeue on NACK.
