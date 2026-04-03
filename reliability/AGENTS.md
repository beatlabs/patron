# reliability

Circuit breaker + retry patterns with OTel metrics.

## Structure

```
reliability/
├── circuitbreaker/breaker.go   # Circuit breaker (closed → open → half-open)
└── retry/retry.go              # Retry with configurable attempts + delay
```

## Code map

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `CircuitBreaker` | struct | `circuitbreaker/breaker.go:72` | State machine: closed/open/half-open |
| `Setting` | struct | `circuitbreaker/breaker.go:57` | FailureThreshold, RetryTimeout, RetrySuccessThreshold, MaxRetryExecutionThreshold |
| `Action` | func | `circuitbreaker/breaker.go:69` | `func() (any, error)` — wrapped operation |
| `(*CircuitBreaker).Execute` | method | `circuitbreaker/breaker.go:129` | Executes action through circuit breaker |
| `OpenError` | struct | `circuitbreaker/breaker.go:17` | Returned when circuit is open |
| `Retry` | struct | `retry/retry.go:13` | Configurable retry with attempts + delay |
| `retry.New` | func | `retry/retry.go:19` | `New(attempts, delay)` |
| `(Retry).Execute` | method | `retry/retry.go:28` | Retries action up to N times |

## Conventions

- Circuit breaker tracks state transitions via OTel metrics (opened/closed counters).
- `Execute` takes `Action func() (any, error)` — returns first success or last error.
- `OpenError` is a sentinel type — check with `errors.As`.
- Used by `client/http` via options for automatic circuit breaking on HTTP calls.
