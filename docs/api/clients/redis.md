# Redis client

Thin wrapper around go-redis/v9 with redisotel tracing and common options.

- Package: `github.com/beatlabs/patron/client/redis`
- Upstream: `github.com/redis/go-redis/v9` with `github.com/redis/go-redis/extra/redisotel/v9`

## Usage

```go
cli, err := redis.New(&redis.Options{Addr: "localhost:6379"})
err = cli.Ping(ctx).Err()
```

- OTEL tracing/metrics auto-enabled via `redisotel`. Pass your own `*redis.Client` if needed.
