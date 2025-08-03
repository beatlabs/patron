# Patron 🏛️

![CI](https://github.com/beatlabs/patron/workflows/CI/badge.svg) [![codecov](https://codecov.io/gh/beatlabs/patron/graph/badge.svg?token=sxY15rXW1X)](https://codecov.io/gh/beatlabs/patron) [![Go Report Card](https://goreportcard.com/badge/github.com/beatlabs/patron)](https://goreportcard.com/report/github.com/beatlabs/patron) [![GoDoc](https://godoc.org/github.com/beatlabs/patron?status.svg)](https://godoc.org/github.com/beatlabs/patron) ![GitHub release](https://img.shields.io/github/release/beatlabs/patron.svg) [![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fbeatlabs%2Fpatron.svg?type=shield&issueType=license)](https://app.fossa.com/projects/git%2Bgithub.com%2Fbeatlabs%2Fpatron?ref=badge_shield&issueType=license)

**A comprehensive Go framework for building production-ready microservices with built-in observability, reliability, and best practices.**

Originally created by [Sotiris Mantzaris](https://github.com/mantzas), now maintained by [Beat Engineering](https://thebeat.co).

> **Etymology**: `Patron` is French for "template" or "pattern", but also means "boss" - which we discovered later (no pun intended)! 😄

## ✨ Features

- 🚀 **Production-Ready**: Built-in health checks, metrics, and graceful shutdown
- 🔍 **Full Observability**: OpenTelemetry integration for tracing, metrics, and structured logging
- 🛡️ **Reliability Patterns**: Circuit breakers, retries, and timeouts out of the box
- 🔌 **Multi-Protocol Support**: HTTP, gRPC, and message queues (Kafka, RabbitMQ, SQS)
- 🗄️ **Database Integration**: Redis, MongoDB, MySQL, Elasticsearch with instrumentation
- ⚡ **High Performance**: Optimized for low latency and high throughput
- 🧩 **Modular Architecture**: Component-based design for flexibility
- 📊 **Built-in Monitoring**: Automatic metrics collection and health endpoints

## 🚀 Quick Start

### Prerequisites

- **Go 1.24+** (required)
- Docker (for running examples with dependencies)

### Installation

```bash
go mod init your-service
go get github.com/beatlabs/patron
```

### Hello World Service

```go
package main

import (
    "context"
    "net/http"
    "log/slog"

    "github.com/beatlabs/patron"
    patronhttp "github.com/beatlabs/patron/component/http"
    "github.com/beatlabs/patron/component/http/router"
)

func main() {
    // Create service
    service, err := patron.New("hello-service", "1.0.0")
    if err != nil {
        slog.Error("Failed to create service", "error", err)
        return
    }

    // Create HTTP component
    httpCmp, err := createHTTPComponent()
    if err != nil {
        slog.Error("Failed to create HTTP component", "error", err)
        return
    }

    // Run service with components
    err = service.Run(context.Background(), httpCmp)
    if err != nil {
        slog.Error("Service failed", "error", err)
    }
}

func createHTTPComponent() (patron.Component, error) {
    // Define handler
    handler := func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Hello, World!"))
    }

    // Create routes
    var routes patronhttp.Routes
    routes.Append(patronhttp.NewRoute("GET /hello", handler))
    rr, err := routes.Result()
    if err != nil {
        return nil, err
    }

    // Create router
    rt, err := router.New(router.WithRoutes(rr...))
    if err != nil {
        return nil, err
    }

    return patronhttp.New(rt)
}
```

### Run the Service

```bash
go run main.go
```

Your service will be available at:

- **Application**: `http://localhost:50000/hello`
- **Health Check**: `http://localhost:50000/alive`
- **Readiness**: `http://localhost:50000/ready`
- **Metrics**: `http://localhost:50000/metrics`

## 📚 Core Concepts

### Service Architecture

Patron follows a **component-based architecture** where:

- **Service**: The main orchestrator that manages components and handles lifecycle
- **Components**: Pluggable modules that handle specific protocols (HTTP, gRPC, messaging)
- **Clients**: Instrumented clients for external services (databases, message queues, APIs)
- **Observability**: Built-in tracing, metrics, and logging with OpenTelemetry

```go
// Service creation with options
service, err := patron.New("my-service", "1.0.0",
    patron.WithJSONLogger(),                    // JSON structured logging
    patron.WithLogFields(slog.String("env", "prod")), // Additional log fields
    patron.WithSIGHUP(reloadConfigHandler),     // Custom signal handling
)

// Run with multiple components
err = service.Run(ctx, httpComponent, grpcComponent, kafkaComponent)
```

### Component Types

| Component | Purpose | Use Cases |
|-----------|---------|-----------|
| **HTTP** | REST APIs, webhooks | Web APIs, health checks, admin interfaces |
| **gRPC** | High-performance RPC | Service-to-service communication |
| **Kafka** | Event streaming | Event sourcing, real-time analytics |
| **AMQP** | Message queuing | Task queues, reliable messaging |
| **SQS** | AWS message queuing | Cloud-native messaging |

## 🔧 Components Guide

### HTTP Component

Create REST APIs with middleware, routing, and automatic instrumentation:

```go
import (
    patronhttp "github.com/beatlabs/patron/component/http"
    "github.com/beatlabs/patron/component/http/router"
    "github.com/beatlabs/patron/component/http/middleware"
)

func createHTTPComponent() (patron.Component, error) {
    // Define handlers
    getUserHandler := func(w http.ResponseWriter, r *http.Request) {
        userID := r.PathValue("id")
        // ... business logic
        json.NewEncoder(w).Encode(map[string]string{"id": userID})
    }

    createUserHandler := func(w http.ResponseWriter, r *http.Request) {
        // ... create user logic
        w.WriteHeader(http.StatusCreated)
    }

    // Create routes with middleware
    var routes patronhttp.Routes
    routes.Append(patronhttp.NewRoute("GET /users/{id}", getUserHandler,
        patronhttp.WithMiddleware(middleware.NewAuth("api-key"))))
    routes.Append(patronhttp.NewRoute("POST /users", createUserHandler))

    rr, err := routes.Result()
    if err != nil {
        return nil, err
    }

    // Create router with global middleware
    rt, err := router.New(
        router.WithRoutes(rr...),
        router.WithMiddleware(middleware.NewLogging()),
        router.WithMiddleware(middleware.NewRecovery()),
    )
    if err != nil {
        return nil, err
    }

    return patronhttp.New(rt)
}
```

### gRPC Component

High-performance RPC services with automatic instrumentation:

```go
import (
    "github.com/beatlabs/patron/component/grpc"
    "google.golang.org/grpc"
)

func createGRPCComponent() (patron.Component, error) {
    // Create gRPC component
    cmp, err := grpc.New(9090,
        grpc.WithServerOptions(grpc.MaxRecvMsgSize(1024*1024)), // 1MB max message
        grpc.WithReflection(), // Enable reflection for debugging
    )
    if err != nil {
        return nil, err
    }

    // Register your service
    pb.RegisterUserServiceServer(cmp.Server(), &userServiceImpl{})

    return cmp, nil
}

type userServiceImpl struct {
    pb.UnimplementedUserServiceServer
}

func (s *userServiceImpl) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    // Automatic tracing and metrics collection
    log.FromContext(ctx).Info("GetUser called", "user_id", req.Id)

    // ... business logic
    return &pb.User{Id: req.Id, Name: "John Doe"}, nil
}
```

### Messaging Components

#### Kafka Consumer

Process events from Kafka topics with automatic offset management:

```go
import (
    "github.com/beatlabs/patron/component/kafka"
)

func createKafkaComponent() (patron.Component, error) {
    return kafka.New("my-consumer-group", []string{"user-events"},
        kafka.WithBrokers([]string{"localhost:9092"}),
        kafka.WithProcessor(processUserEvent),
        kafka.WithRetry(3, time.Second*5),
    )
}

func processUserEvent(ctx context.Context, msg kafka.Message) error {
    log.FromContext(ctx).Info("Processing user event",
        "topic", msg.Topic(),
        "partition", msg.Partition(),
        "offset", msg.Offset())

    var event UserEvent
    if err := json.Unmarshal(msg.Body(), &event); err != nil {
        return fmt.Errorf("failed to unmarshal event: %w", err)
    }

    // Process the event
    return handleUserEvent(ctx, event)
}
```

#### AMQP Consumer (RabbitMQ)

Reliable message processing with RabbitMQ:

```go
import (
    "github.com/beatlabs/patron/component/amqp"
)

func createAMQPComponent() (patron.Component, error) {
    return amqp.New("amqp://guest:guest@localhost:5672/", "user-queue",
        amqp.WithProcessor(processMessage),
        amqp.WithRetry(3),
        amqp.WithPrefetchCount(10),
    )
}

func processMessage(ctx context.Context, msg amqp.Message) error {
    log.FromContext(ctx).Info("Processing AMQP message", "body", string(msg.Body()))

    // Process message
    if err := processBusinessLogic(ctx, msg.Body()); err != nil {
        return err // Message will be requeued
    }

    return nil // Message acknowledged
}
```

#### AWS SQS Consumer

Cloud-native message processing with AWS SQS:

```go
import (
    "github.com/beatlabs/patron/component/sqs"
    "github.com/aws/aws-sdk-go-v2/config"
)

func createSQSComponent(ctx context.Context) (patron.Component, error) {
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        return nil, err
    }

    return sqs.New("https://sqs.us-east-1.amazonaws.com/123456789/my-queue", cfg,
        sqs.WithProcessor(processSQSMessage),
        sqs.WithMaxMessages(10),
        sqs.WithWaitTimeSeconds(20),
    )
}

func processSQSMessage(ctx context.Context, msg sqs.Message) error {
    log.FromContext(ctx).Info("Processing SQS message", "messageId", msg.MessageID())

    // Process message
    return handleSQSMessage(ctx, msg.Body())
}
```

## 🗄️ Client Libraries

Patron provides instrumented clients for popular services with automatic tracing and metrics:

### Redis Client

```go
import (
    patronredis "github.com/beatlabs/patron/client/redis"
    "github.com/redis/go-redis/v9"
)

func setupRedis() (*redis.Client, error) {
    return patronredis.New(&redis.Options{
        Addr:     "localhost:6379",
        Password: "", // no password
        DB:       0,  // default DB
    })
}

func useRedis(ctx context.Context, client *redis.Client) error {
    // All operations are automatically traced and measured
    err := client.Set(ctx, "key", "value", time.Hour).Err()
    if err != nil {
        return err
    }

    val, err := client.Get(ctx, "key").Result()
    if err != nil {
        return err
    }

    log.FromContext(ctx).Info("Retrieved value", "value", val)
    return nil
}
```

### MongoDB Client

```go
import (
    patronmongo "github.com/beatlabs/patron/client/mongo"
    "go.mongodb.org/mongo-driver/bson"
)

func setupMongoDB(ctx context.Context) (*mongo.Client, error) {
    return patronmongo.Connect(ctx,
        options.Client().ApplyURI("mongodb://localhost:27017"))
}

func useMongoDB(ctx context.Context, client *mongo.Client) error {
    collection := client.Database("mydb").Collection("users")

    // Insert with automatic tracing
    _, err := collection.InsertOne(ctx, bson.M{"name": "John", "age": 30})
    if err != nil {
        return err
    }

    // Find with automatic tracing
    var result bson.M
    err = collection.FindOne(ctx, bson.M{"name": "John"}).Decode(&result)
    return err
}
```

### AWS Clients (SNS, SQS)

```go
import (
    patronsns "github.com/beatlabs/patron/client/sns"
    patronsqs "github.com/beatlabs/patron/client/sqs"
    "github.com/aws/aws-sdk-go-v2/config"
)

func setupAWSClients(ctx context.Context) error {
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        return err
    }

    // SNS client with instrumentation
    snsClient := patronsns.NewFromConfig(cfg)

    // SQS client with instrumentation
    sqsClient := patronsqs.NewFromConfig(cfg)

    // Use clients - all operations are automatically traced
    _, err = snsClient.Publish(ctx, &sns.PublishInput{
        TopicArn: aws.String("arn:aws:sns:us-east-1:123456789:my-topic"),
        Message:  aws.String("Hello from Patron!"),
    })

    return err
}
```

## 🔍 Observability

Patron provides comprehensive observability out of the box with OpenTelemetry integration.

### Structured Logging

```go
import (
    "github.com/beatlabs/patron/observability/log"
)

func businessLogic(ctx context.Context) {
    // Get logger from context (includes trace correlation)
    logger := log.FromContext(ctx)

    logger.Info("Processing request",
        "user_id", "12345",
        "operation", "create_order")

    logger.Error("Failed to process",
        "error", err,
        "retry_count", 3)
}

// Configure JSON logging
service, err := patron.New("my-service", "1.0.0",
    patron.WithJSONLogger(),
    patron.WithLogFields(
        slog.String("environment", "production"),
        slog.String("region", "us-east-1"),
    ),
)
```

### Distributed Tracing

Automatic trace propagation across components and clients:

```go
import (
    "github.com/beatlabs/patron/observability/trace"
)

func handleRequest(ctx context.Context) error {
    // Create custom span
    ctx, span := trace.StartSpan(ctx, "business-operation")
    defer span.End()

    // Add attributes
    span.SetAttributes(
        attribute.String("user.id", "12345"),
        attribute.String("operation.type", "create"),
    )

    // Call external service (automatically traced)
    err := callExternalAPI(ctx)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }

    span.SetStatus(codes.Ok, "success")
    return nil
}
```

### Custom Metrics

```go
import (
    patronmetric "github.com/beatlabs/patron/observability/metric"
    "go.opentelemetry.io/otel/attribute"
)

var (
    requestCounter = patronmetric.Int64Counter("requests", "total_requests", "Total number of requests")
    responseTime   = patronmetric.Int64Histogram("response_time", "response_time_ms", "Response time in milliseconds", "ms")
)

func recordMetrics(ctx context.Context, duration time.Duration, status string) {
    attrs := []attribute.KeyValue{
        attribute.String("status", status),
        attribute.String("endpoint", "/api/users"),
    }

    requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
    responseTime.Record(ctx, duration.Milliseconds(), metric.WithAttributes(attrs...))
}
```

## 🛡️ Reliability Patterns

### Circuit Breaker

Protect your service from cascading failures:

```go
import (
    "github.com/beatlabs/patron/reliability/circuitbreaker"
)

func setupCircuitBreaker() *circuitbreaker.CircuitBreaker {
    return circuitbreaker.New(
        circuitbreaker.WithFailureThreshold(5),     // Open after 5 failures
        circuitbreaker.WithSuccessThreshold(3),     // Close after 3 successes
        circuitbreaker.WithTimeout(time.Second*30), // Half-open timeout
    )
}

func callExternalService(ctx context.Context, cb *circuitbreaker.CircuitBreaker) error {
    return cb.Execute(ctx, func(ctx context.Context) error {
        // Your external service call
        return httpClient.Get(ctx, "https://api.example.com/data")
    })
}
```

### Retry Logic

Automatic retries with exponential backoff:

```go
import (
    "github.com/beatlabs/patron/reliability/retry"
)

func callWithRetry(ctx context.Context) error {
    return retry.Do(ctx, func(ctx context.Context) error {
        // Your operation that might fail
        return callUnreliableAPI(ctx)
    },
        retry.WithMaxAttempts(3),
        retry.WithBackoff(retry.ExponentialBackoff(time.Second, time.Minute)),
        retry.WithRetryableErrors([]error{ErrTemporary, ErrRateLimited}),
    )
}
```

## 🚀 Production Deployment

### Environment Configuration

```bash
# Service configuration
export PATRON_LOG_LEVEL=info
export PATRON_HTTP_DEFAULT_PORT=8080

# OpenTelemetry configuration
export OTEL_SERVICE_NAME=my-service
export OTEL_SERVICE_VERSION=1.0.0
export OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:14268/api/traces

# Application-specific
export DATABASE_URL=postgres://user:pass@localhost/db
export REDIS_URL=redis://localhost:6379
```

### Docker Deployment

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o service ./cmd/service

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/service .

EXPOSE 8080
CMD ["./service"]
```

### Health Checks

Patron automatically provides health check endpoints:

- **`/alive`**: Liveness probe - service is running
- **`/ready`**: Readiness probe - service is ready to accept traffic
- **`/metrics`**: Prometheus metrics endpoint

```yaml
# Kubernetes deployment example
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-service
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: service
        image: my-service:latest
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /alive
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

## 📖 Examples

Check out the [examples directory](./examples) for complete working examples:

- **[HTTP Service](./examples/service/http.go)**: REST API with middleware
- **[gRPC Service](./examples/service/grpc.go)**: High-performance RPC
- **[Kafka Consumer](./examples/service/kafka.go)**: Event processing
- **[AMQP Consumer](./examples/service/amqp.go)**: Message queue processing
- **[SQS Consumer](./examples/service/sqs.go)**: AWS SQS integration

### Running Examples

```bash
# Start dependencies
make deps-start

# Run the example service
make example-service

# In another terminal, run the client
make example-client

# Stop dependencies
make deps-stop
```

## 🤝 Contributing

We welcome contributions! Please see our [Contribution Guidelines](docs/ContributionGuidelines.md) for details.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/beatlabs/patron.git
cd patron

# Install dependencies and run tests
make test

# Run integration tests
make testint

# Run linting
make lint

# Format code
make fmt
```

## 📚 Documentation

- [Code of Conduct](docs/CodeOfConduct.md)
- [Contribution Guidelines](docs/ContributionGuidelines.md)
- [Acknowledgments](docs/ACKNOWLEDGMENTS.md)
- [Breaking Changes](BREAKING.md)

## 📊 Code Coverage

![Code coverage](https://codecov.io/gh/beatlabs/patron/graphs/icicle.svg?token=sxY15rXW1X)

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

Special thanks to all [contributors](docs/ACKNOWLEDGMENTS.md) who have helped make Patron better!
