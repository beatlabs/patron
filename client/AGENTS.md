# client

Client wrappers for external services. All follow functional options + OTel instrumentation pattern.

## Structure

```
client/
├── http/              # HTTP client with circuit breaker + tracing
│   ├── http.go       # TracedClient wrapping Client interface
│   ├── option.go     # OptionFunc (circuit breaker, transport, timeout)
│   └── encoding/json/ # Response JSON decoder
├── grpc/grpc.go       # gRPC client with tracing interceptors
├── kafka/kafka.go     # Kafka producer (franz-go) with sync/async send + metrics
├── amqp/              # RabbitMQ publisher with tracing
│   ├── amqp.go       # Publisher with connection management
│   └── option.go     # Exchange, timeout options
├── sqs/sqs.go         # AWS SQS publisher (SDK v2)
├── sns/sns.go         # AWS SNS publisher (SDK v2)
├── sql/sql.go         # SQL client wrapping database/sql with tracing (518 LOC)
├── redis/redis.go     # Redis client (go-redis/v9)
├── mongo/             # MongoDB client with metrics
│   ├── mongo.go      # Client with tracing
│   └── metric.go     # Operation metrics
├── es/                # Elasticsearch client
│   ├── elasticsearch.go # Client wrapper
│   └── instrumentation.go # OTel transport
└── mqtt/              # MQTT publisher
    ├── publisher.go   # Publisher with metrics
    └── metric.go      # Publish metrics
```

## Conventions

- **Constructor**: `New(required, opts...) (*Client, error)` — always functional options, always returns error.
- **OTel instrumentation**: every client wraps operations with spans; attributes follow `observability.ClientAttribute`.
- **Correlation**: clients propagate `X-Correlation-Id` header via context (see `correlation/` package).
- **Circuit breaker**: HTTP client supports optional `circuitbreaker.CircuitBreaker` via options.
- **Metrics**: clients with high-throughput operations (Kafka, MQTT, Mongo) have dedicated `metric.go` files.
- **AWS clients**: use AWS SDK v2 (`aws-sdk-go-v2`); require `aws.Config` in constructor.

## Code map

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `http.TracedClient` | struct | `http/http.go:24` | HTTP client with tracing + optional circuit breaker |
| `kafka.Producer` | struct | `kafka/kafka.go:43` | Kafka producer with `Send`/`SendAsync`/`Close` |
| `sql.DB` (wrapped) | — | `sql/sql.go` | Wraps `database/sql` with tracing (largest client: 518 LOC) |

## Adding a new client

1. Create `client/<name>/<name>.go`
2. Define struct + `New(required, opts...) (*Client, error)`
3. Add `OptionFunc` in `option.go`
4. Wrap operations with OTel spans using `observability/trace`
5. Propagate correlation ID from context
6. Add metrics in `metric.go` if high-throughput
