# Patron API Documentation

This document provides comprehensive API documentation for the Patron microservices framework.

## Table of Contents

- [Core Service API](#core-service-api)
- [HTTP Component API](#http-component-api)
- [gRPC Component API](#grpc-component-api)
- [Messaging Components API](#messaging-components-api)
- [Client Libraries API](#client-libraries-api)
- [Observability API](#observability-api)
- [Reliability API](#reliability-api)

## Core Service API

### Service Creation

```go
func New(name, version string, options ...OptionFunc) (*Service, error)
```

Creates a new Patron service instance.

**Parameters:**

- `name` (string): Service name (required)
- `version` (string): Service version (optional, defaults to "dev")
- `options` (...OptionFunc): Configuration options

**Returns:**

- `*Service`: Service instance
- `error`: Error if creation fails

**Example:**

```go
service, err := patron.New("user-service", "1.2.3",
    patron.WithJSONLogger(),
    patron.WithLogFields(slog.String("env", "prod")),
    patron.WithSIGHUP(reloadHandler),
)
```

### Service Options

#### WithJSONLogger()

Enables JSON structured logging instead of text format.

```go
func WithJSONLogger() OptionFunc
```

#### WithLogFields(attrs ...slog.Attr)

Adds additional log fields to all log entries.

```go
func WithLogFields(attrs ...slog.Attr) OptionFunc
```

**Parameters:**

- `attrs` (...slog.Attr): Log attributes to add

**Example:**

```go
patron.WithLogFields(
    slog.String("environment", "production"),
    slog.String("region", "us-east-1"),
    slog.Int("version", 2),
)
```

#### WithSIGHUP(handler func())

Sets a custom SIGHUP signal handler.

```go
func WithSIGHUP(handler func()) OptionFunc
```

**Parameters:**

- `handler` (func()): Function to call on SIGHUP signal

### Service Methods

#### Run(ctx context.Context, components ...Component) error

Starts the service with the provided components.

```go
func (s *Service) Run(ctx context.Context, components ...Component) error
```

**Parameters:**

- `ctx` (context.Context): Context for service lifecycle
- `components` (...Component): Components to run

**Returns:**

- `error`: Error if service fails to start or run

## HTTP Component API

### Component Creation

```go
func New(router router.Router, options ...OptionFunc) (patron.Component, error)
```

Creates a new HTTP component.

**Parameters:**

- `router` (router.Router): HTTP router instance
- `options` (...OptionFunc): Configuration options

### Router Creation

```go
func New(options ...OptionFunc) (Router, error)
```

Creates a new HTTP router.

#### Router Options

##### WithRoutes(routes ...Route)

Adds routes to the router.

```go
func WithRoutes(routes ...Route) OptionFunc
```

##### WithMiddleware(middlewares ...MiddlewareFunc)

Adds global middleware to all routes.

```go
func WithMiddleware(middlewares ...MiddlewareFunc) OptionFunc
```

### Route Creation

```go
func NewRoute(pattern string, handler http.HandlerFunc, options ...RouteOptionFunc) Route
```

Creates a new HTTP route.

**Parameters:**

- `pattern` (string): HTTP method and path pattern (e.g., "GET /users/{id}")
- `handler` (http.HandlerFunc): Request handler
- `options` (...RouteOptionFunc): Route-specific options

#### Route Options

##### WithMiddleware(middlewares ...MiddlewareFunc)

Adds middleware to a specific route.

```go
func WithMiddleware(middlewares ...MiddlewareFunc) RouteOptionFunc
```

### Routes Collection

```go
type Routes struct{}

func (r *Routes) Append(route Route)
func (r *Routes) Result() ([]Route, error)
```

**Methods:**

- `Append(route Route)`: Adds a route to the collection
- `Result() ([]Route, error)`: Returns all routes and validates them

### Middleware

#### Built-in Middleware

##### NewLogging()

Logs HTTP requests and responses.

```go
func NewLogging() MiddlewareFunc
```

##### NewRecovery()

Recovers from panics and returns 500 status.

```go
func NewRecovery() MiddlewareFunc
```

##### NewAuth(apiKey string)

Validates API key authentication.

```go
func NewAuth(apiKey string) MiddlewareFunc
```

## gRPC Component API

### Component Creation

```go
func New(port int, options ...OptionFunc) (patron.Component, error)
```

Creates a new gRPC component.

**Parameters:**

- `port` (int): Port to listen on
- `options` (...OptionFunc): Configuration options

#### gRPC Options

##### WithServerOptions(opts ...grpc.ServerOption)

Adds gRPC server options.

```go
func WithServerOptions(opts ...grpc.ServerOption) OptionFunc
```

##### WithReflection()

Enables gRPC reflection for debugging.

```go
func WithReflection() OptionFunc
```

### Component Methods

#### Server() *grpc.Server

Returns the underlying gRPC server for service registration.

```go
func (c *Component) Server() *grpc.Server
```

**Example:**

```go
grpcComponent, err := grpc.New(9090, grpc.WithReflection())
if err != nil {
    return err
}

// Register your service
pb.RegisterUserServiceServer(grpcComponent.Server(), &userServiceImpl{})
```

## Messaging Components API

### Kafka Component

#### Component Creation

```go
func New(group string, topics []string, options ...OptionFunc) (patron.Component, error)
```

**Parameters:**

- `group` (string): Consumer group name
- `topics` ([]string): Topics to consume from
- `options` (...OptionFunc): Configuration options

#### Kafka Options

##### WithBrokers(brokers []string)

Sets Kafka broker addresses.

```go
func WithBrokers(brokers []string) OptionFunc
```

##### WithProcessor(processor ProcessorFunc)

Sets the message processor function.

```go
func WithProcessor(processor ProcessorFunc) OptionFunc
```

##### WithRetry(count int, delay time.Duration)

Configures retry behavior.

```go
func WithRetry(count int, delay time.Duration) OptionFunc
```

#### Message Interface

```go
type Message interface {
    Topic() string
    Partition() int32
    Offset() int64
    Key() []byte
    Body() []byte
    Headers() map[string]string
    Span() trace.Span
}
```

#### Processor Function

```go
type ProcessorFunc func(ctx context.Context, msg Message) error
```

### AMQP Component

#### Component Creation

```go
func New(url, queue string, options ...OptionFunc) (patron.Component, error)
```

**Parameters:**

- `url` (string): AMQP connection URL
- `queue` (string): Queue name to consume from
- `options` (...OptionFunc): Configuration options

#### AMQP Options

##### WithProcessor(processor ProcessorFunc)

Sets the message processor function.

##### WithRetry(count int)

Sets retry count for failed messages.

##### WithPrefetchCount(count int)

Sets prefetch count for message consumption.

### SQS Component

#### Component Creation

```go
func New(queueURL string, cfg aws.Config, options ...OptionFunc) (patron.Component, error)
```

**Parameters:**

- `queueURL` (string): SQS queue URL
- `cfg` (aws.Config): AWS configuration
- `options` (...OptionFunc): Configuration options

#### SQS Options

##### WithProcessor(processor ProcessorFunc)

Sets the message processor function.

##### WithMaxMessages(max int32)

Sets maximum messages to receive per poll.

##### WithWaitTimeSeconds(seconds int32)

Sets long polling wait time.

## Client Libraries API

### Redis Client

#### Client Creation

```go
func New(opt *redis.Options) (*redis.Client, error)
```

Creates a Redis client with automatic instrumentation.

**Parameters:**

- `opt` (*redis.Options): Redis connection options

**Returns:**

- `*redis.Client`: Instrumented Redis client
- `error`: Error if creation fails

**Example:**

```go
client, err := patronredis.New(&redis.Options{
    Addr:     "localhost:6379",
    Password: "secret",
    DB:       0,
})
```

### MongoDB Client

#### Client Creation

```go
func Connect(ctx context.Context, opts ...*options.ClientOptions) (*mongo.Client, error)
```

Creates a MongoDB client with automatic instrumentation.

**Parameters:**

- `ctx` (context.Context): Connection context
- `opts` (...*options.ClientOptions): MongoDB client options

**Returns:**

- `*mongo.Client`: Instrumented MongoDB client
- `error`: Error if connection fails

### AWS Clients

#### SNS Client

```go
func NewFromConfig(cfg aws.Config, optFns ...func(*sns.Options)) *sns.Client
```

Creates an SNS client with automatic instrumentation.

#### SQS Client

```go
func NewFromConfig(cfg aws.Config, optFns ...func(*sqs.Options)) *sqs.Client
```

Creates an SQS client with automatic instrumentation.

## Observability API

### Logging

#### FromContext(ctx context.Context) *slog.Logger

Gets a logger from context with trace correlation.

```go
func FromContext(ctx context.Context) *slog.Logger
```

**Example:**

```go
logger := log.FromContext(ctx)
logger.Info("Processing request", "user_id", userID)
```

#### ErrorAttr(err error) slog.Attr

Creates an error attribute for structured logging.

```go
func ErrorAttr(err error) slog.Attr
```

### Tracing

#### StartSpan(ctx context.Context, name string) (context.Context, trace.Span)

Creates a new trace span.

```go
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span)
```

**Parameters:**

- `ctx` (context.Context): Parent context
- `name` (string): Span name

**Returns:**

- `context.Context`: Context with span
- `trace.Span`: Created span

#### Setup(serviceName string, sampler trace.Sampler, exporter trace.SpanExporter) *trace.TracerProvider

Sets up tracing configuration.

### Metrics

#### Int64Counter(name, description string) metric.Int64Counter

Creates an integer counter metric.

```go
func Int64Counter(name, description string) metric.Int64Counter
```

#### Int64Histogram(name, description, unit string) metric.Int64Histogram

Creates an integer histogram metric.

```go
func Int64Histogram(name, description, unit string) metric.Int64Histogram
```

#### Float64Gauge(name, description string) metric.Float64ObservableGauge

Creates a float64 gauge metric.

```go
func Float64Gauge(name, description string) metric.Float64ObservableGauge
```

## Reliability API

### Circuit Breaker

#### Creation

```go
func New(options ...OptionFunc) *CircuitBreaker
```

Creates a new circuit breaker.

#### Options

##### WithFailureThreshold(threshold int)

Sets failure threshold to open circuit.

##### WithSuccessThreshold(threshold int)

Sets success threshold to close circuit.

##### WithTimeout(timeout time.Duration)

Sets timeout for half-open state.

#### Methods

##### Execute(ctx context.Context, fn func(context.Context) error) error

Executes function with circuit breaker protection.

### Retry

#### Do(ctx context.Context, fn func(context.Context) error, options ...OptionFunc) error

Executes function with retry logic.

#### Options

##### WithMaxAttempts(attempts int)

Sets maximum retry attempts.

##### WithBackoff(backoff BackoffFunc)

Sets backoff strategy.

##### WithRetryableErrors(errors []error)

Sets which errors should trigger retries.

#### Backoff Strategies

##### ExponentialBackoff(initial, max time.Duration) BackoffFunc

Creates exponential backoff strategy.

##### FixedBackoff(delay time.Duration) BackoffFunc

Creates fixed delay backoff strategy.
