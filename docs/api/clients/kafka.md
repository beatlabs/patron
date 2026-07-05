# Kafka client (producer)

Instrumented producer for Kafka using franz-go, with OpenTelemetry hooks, correlation propagation, and publish metrics.

- Package: `github.com/beatlabs/patron/client/kafka`
- Producer: `Producer` with synchronous (`Send`) and asynchronous (`SendAsync`) sends

## Create a producer

```go
producer, err := clientkafka.New([]string{"localhost:9092"})
if err != nil {
    // handle error
}
defer producer.Close()
```

`New` validates broker addresses and automatically configures:

- franz-go seed brokers
- OpenTelemetry tracing and metrics hooks
- slog-backed franz-go logging

## Passing franz-go options

Use `WithKafkaOptions` to pass raw franz-go `kgo.Opt` values.

```go
producer, err := clientkafka.New(
    []string{"localhost:9092"},
    clientkafka.WithKafkaOptions(
        kgo.ClientID("orders-producer"),
        kgo.RequiredAcks(kgo.AllISRAcks()),
    ),
)
if err != nil {
    // handle error
}
defer producer.Close()
```

This keeps the Patron client constructor on the functional-options pattern while preserving access to franz-go's full option surface.

## Sync send

```go
records, err := producer.Send(ctx, &kgo.Record{
    Topic: "orders",
    Value: []byte("payload"),
})
if err != nil {
    // handle error
}

_ = records
```

## Batch send

```go
records, err := producer.Send(ctx,
    &kgo.Record{Topic: "orders", Value: []byte("one")},
    &kgo.Record{Topic: "orders", Value: []byte("two")},
)
if err != nil {
    // handle error
}

_ = records
```

## Async send

```go
chErr := make(chan error, 1)

producer.SendAsync(ctx, &kgo.Record{
    Topic: "orders",
    Value: []byte("payload"),
}, chErr)

select {
case err := <-chErr:
    // handle producer error
default:
}
```

## Common methods

- `Send(ctx context.Context, records ...*kgo.Record) ([]*kgo.Record, error)` sends one or more records synchronously.
- `SendAsync(ctx context.Context, record *kgo.Record, chErr chan<- error)` sends one record asynchronously and reports producer errors on `chErr`.
- `Close()` flushes buffered records and shuts down the producer.

See `client/kafka` and `examples/client/main.go` for more details.
