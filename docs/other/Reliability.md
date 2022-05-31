# Reliability

The reliability package contains the following implementations:

- Circuit Breaker Pattern
- Retry Pattern

## Circuit Breaker Pattern

The circuit breaker supports a half-open state which allows probing for successful responses in order to close the circuit again.  
Every aspect of the circuit breaker is configurable via its settings.

The action to be executed has the signature type of:

```go
type Action func() (interface{}, error)
```

and its following aspects can be configured:

- The threshold for the circuit to open
- The timeout after which we set the state to half-open and allow retries
- The threshold of retry successes which returns the state to open
- The threshold of how many retry executions are allowed when the status is half-open

## Retry Pattern

Retry accepts a function with the following signature:

```go
type Action func() (interface{}, error)
``` 

and retries the action for a configurable amount of retries with a specific fixed time interval between them.