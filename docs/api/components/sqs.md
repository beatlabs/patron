# SQS component (AWS SQS consumer)

Consumes messages from AWS SQS with long polling, visibility timeouts, retries, and observability.

- Package: `github.com/beatlabs/patron/component/sqs`
- Type: component implementing `Run(ctx context.Context) error`
- Built-ins: OpenTelemetry tracing/metrics, correlation propagation, periodic queue stats, batch ack/nack helpers

## Quick start

```go
import (
  patronsqs "github.com/beatlabs/patron/component/sqs"
)

proc := func(ctx context.Context, batch patronsqs.Batch) {
  for _, m := range batch.Messages() {
    // handle m.Body() or use the raw message via m.Message()
    _ = m.ACK() // or m.NACK()
  }
}

cmp, err := patronsqs.New("sqs-cmp", "queue-name", sqsClient, proc,
  patronsqs.WithPollWaitSeconds(5), // long polling
)
// add cmp to a Service and Run
```

See the runnable example: `examples/service/sqs.go`.

## Options

```go
// Max messages per receive (1..10)
patronsqs.WithMaxMessages(n)

// Long polling wait seconds (0..20)
patronsqs.WithPollWaitSeconds(seconds)

// Visibility timeout in seconds (0..43200)
patronsqs.WithVisibilityTimeout(seconds)

// Stats interval for queue metrics
patronsqs.WithQueueStatsInterval(interval)

// Retry settings for transient failures
patronsqs.WithRetries(count)
patronsqs.WithRetryWait(interval)

// Queue owner AWS account ID (for cross-account usage)
patronsqs.WithQueueOwner(ownerID)
```

Notes

- Batching: a batch is built from a receive call; you can call `Batch.ACK()`/`Batch.NACK()` to ack/nack multiple messages; returns any failures and a joined error.
- Each message has `Context()` with a logger and correlation ID and a `Span()`; calling `ACK()`/`NACK()` ends the span and records success/error.
- `WithPollWaitSeconds` enables long polling; prefer >0 to reduce empty receives and costs.
- `WithVisibilityTimeout` should exceed your processing time to avoid redeliveries.
- Stats use `GetQueueAttributes` to publish gauges for available/delayed/invisible counts.
