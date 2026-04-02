# cache

Cache interfaces + implementations. Context-first API (since v0.75.0).

## Structure

```
cache/
├── cache.go     # Cache and TTLCache interfaces + metrics
├── metric.go    # Cache operation metrics (hit/miss/set/remove)
├── lru/lru.go   # LRU cache (hashicorp/golang-lru/v2) with OTel metrics
└── redis/        # Redis-backed cache (go-redis/v9, integration-tested)
```

## Code map

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `Cache[K, V]` | interface | `cache.go:10` | `Get(ctx, key)`, `Set(ctx, key, val)`, `Remove(ctx, key)`, `Purge(ctx)` |
| `TTLCache[K, V]` | interface | `cache.go:22` | Extends `Cache` + `SetTTL(ctx, key, val, ttl)` |
| `lru.Cache` | struct | `lru/lru.go:20` | Generic LRU wrapping hashicorp/golang-lru with metrics |
| `lru.New` | func | `lru/lru.go:27` | `New[K, V](name, size)` — creates LRU cache |
| `lru.NewWithEvict` | func | `lru/lru.go:38` | LRU with eviction callback |

## Conventions

- All methods take `context.Context` as first param (breaking change from v0.75.0).
- Implementations report metrics via `cache/metric.go` (hit/miss counters).
- LRU is generic: `Cache[K comparable, V any]`.
- Redis implementation requires integration tests (`//go:build integration`).

## Adding a new implementation

1. Create `cache/<name>/<name>.go`
2. Implement `Cache[K, V]` or `TTLCache[K, V]` interface
3. Integrate metrics from `cache/metric.go`
4. Add integration tests if external dependency
