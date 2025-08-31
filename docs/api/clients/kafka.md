# Kafka client (producer)

Instrumented producers for Kafka using Sarama, with OpenTelemetry tracing, correlation propagation, and metrics.

- Package: `github.com/beatlabs/patron/client/kafka`
- Producers: synchronous (SyncProducer) and asynchronous (AsyncProducer)

## Sync producer example

```go
builder := clientkafka.New([]string{"localhost:9092"}, cfg)
prod, err := builder.Create()
if err != nil { /* handle */ }

defer prod.Close()

msg := &sarama.ProducerMessage{Topic: "orders", Value: sarama.StringEncoder("payload")}
partition, offset, err := prod.Send(ctx, msg)
```

## Async producer example

```go
prod, chErr, err := clientkafka.New([]string{"localhost:9092"}, cfg).CreateAsync()
if err != nil { /* handle */ }

go func() { for err := range chErr { /* handle */ } }()

defer prod.Close()

_ = prod.Send(ctx, &sarama.ProducerMessage{Topic: "orders", Value: sarama.StringEncoder("payload")})
```

See `client/kafka` package and `examples/client/main.go` for more details.
