# Caching

The caching package contains two interfaces for caching implementations,  
one simple and one supporting setting TTL.

## Interfaces

Cache interface:

```go
type Cache interface {
    Get(key string) (interface{}, bool, error)
    Purge() error
    Remove(key string) error
    Set(key string, value interface{}) error
}
```

TTLCache interface:

```go
type TTLCache interface {
    Cache
    SetTTL(key string, value interface{}, ttl time.Duration) error
}
```

## Implementations

Subpackages contain concrete implementations of the aforementioned interfaces:

- `lru` which contains an in-memory LRU cache implementation of the `Cache` interface
- `redis` which contains a Redis-based cache implementation of the `TTLCache` interface