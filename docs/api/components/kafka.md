# Kafka component (consumer)

Consume messages from Kafka using consumer groups with batching, retries, commit control, and built-in observability.

- Package: `github.com/beatlabs/patron/component/kafka`
- Type: component implementing `Run(ctx context.Context) error`
- Built-ins: OpenTelemetry tracing/metrics, correlation propagation, high-watermark lag gauge, failure strategies

## Quick start

```go
import (
  "time"
  patronkafka "github.com/beatlabs/patron/component/kafka"
  "github.com/IBM/sarama"
)

cfg, _ := patronkafka.DefaultConsumerSaramaConfig("kafka-consumer", true)
cfg.Version = sarama.V2_6_0_0 // set to your broker version

proc := func(batch patronkafka.Batch) error {
  for _, m := range batch.Messages() {
    // m.Context() carries a logger, correlation ID and an active span
    // m.Message() gives you the raw *sarama.ConsumerMessage
    _ = m // handle message, deserialize, etc.
  }
  return nil
}

cmp, err := patronkafka.New(
  "orders-consumer",
  "orders-group",
  []string{"localhost:9092"},
  []string{"orders"},
  proc,
  cfg,
  patronkafka.WithBatchSize(10),
  patronkafka.WithBatchTimeout(500*time.Millisecond),
  patronkafka.WithFailureStrategy(patronkafka.SkipStrategy),
)
// add cmp to a Service and Run
```

See the runnable example: `examples/service/kafka.go`.

## Options

```go
// Failure handling when processing a batch errors
patronkafka.WithFailureStrategy(patronkafka.ExitStrategy | patronkafka.SkipStrategy)

// Validate topics exist on the broker at startup
patronkafka.WithCheckTopic()

// Component-level retry on errors (reconnect/backoff)
patronkafka.WithRetries(count)
patronkafka.WithRetryWait(interval)

// Batching controls
patronkafka.WithBatchSize(n)
patronkafka.WithBatchTimeout(d)

// Deduplicate messages within a batch by key (per-partition ordering assumed)
patronkafka.WithBatchMessageDeduplication()

// Commit offsets synchronously after each processed batch
patronkafka.WithCommitSync()

// Hook invoked on new consumer group session (e.g., rebalances)
patronkafka.WithNewSessionCallback(func(sarama.ConsumerGroupSession) error { /* ... */ })
```

Notes

- Sarama config is required. Use `DefaultConsumerSaramaConfig(name, readCommitted)` to start and pin `cfg.Version` to your cluster.
- Each message exposes:
  - `Context()` with logger and correlation ID.
  - `Message()` returning `*sarama.ConsumerMessage`.
  - `Span()` for tracing; success/error is recorded and ended after batch processing.
- Metrics: message status counter (received/processed/errored/skipped), partition lag gauge (high watermark - offset), and consumer error counter.
- Tracing and correlation headers are extracted automatically using OpenTelemetry propagators.
