# Kafka client (producer)

Instrumented producers for Kafka using Sarama, with OpenTelemetry tracing, correlation propagation, and metrics.

- Package: `github.com/beatlabs/patron/client/kafka`
- Producers: synchronous (`SyncProducer`) and asynchronous (`AsyncProducer`)

## Default Sarama config

```go
cfg, err := clientkafka.DefaultProducerSaramaConfig("my-producer", true /* idempotent */)
// Sets ClientID = "<hostname>-my-producer", RequiredAcks = WaitForAll.
// When idempotent is true: Net.MaxOpenRequests = 1, Producer.Idempotent = true.
```

## Sync producer example

```go
builder := clientkafka.New([]string{"localhost:9092"}, cfg)
prod, err := builder.Create()
if err != nil { /* handle */ }

defer prod.Close()

msg := &sarama.ProducerMessage{Topic: "orders", Value: sarama.StringEncoder("payload")}
partition, offset, err := prod.Send(ctx, msg)
```

### Batch send

```go
messages := []*sarama.ProducerMessage{
  {Topic: "orders", Value: sarama.StringEncoder("one")},
  {Topic: "orders", Value: sarama.StringEncoder("two")},
}
err := prod.SendBatch(ctx, messages)
```

## Async producer example

```go
prod, chErr, err := clientkafka.New([]string{"localhost:9092"}, cfg).CreateAsync()
if err != nil { /* handle */ }

go func() { for err := range chErr { /* handle */ } }()

defer prod.Close()

_ = prod.Send(ctx, &sarama.ProducerMessage{Topic: "orders", Value: sarama.StringEncoder("payload")})
```

## Common methods

Both `SyncProducer` and `AsyncProducer` expose:

- `ActiveBrokers() []string` — returns addresses of active brokers.
- `Close() error` — shuts down the producer and flushes buffered messages. Must be called before the producer goes out of scope to avoid leaking memory.

See `client/kafka` package and `examples/client/main.go` for more details.
