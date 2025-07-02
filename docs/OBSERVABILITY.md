# Observability Guide

This guide covers Patron's comprehensive observability features including logging, tracing, and metrics.

## Table of Contents

- [Overview](#overview)
- [Structured Logging](#structured-logging)
- [Distributed Tracing](#distributed-tracing)
- [Metrics Collection](#metrics-collection)
- [Health Checks](#health-checks)
- [Configuration](#configuration)
- [Best Practices](#best-practices)

## Overview

Patron provides built-in observability through OpenTelemetry integration, offering:

- **Structured Logging**: JSON-formatted logs with trace correlation
- **Distributed Tracing**: Automatic trace propagation across components
- **Metrics Collection**: Prometheus-compatible metrics
- **Health Checks**: Built-in health and readiness endpoints

All observability features are automatically configured and require minimal setup.

## Structured Logging

### Basic Logging

```go
import (
    "github.com/beatlabs/patron/observability/log"
)

func handleRequest(ctx context.Context, userID string) {
    logger := log.FromContext(ctx)
    
    // Info logging with structured fields
    logger.Info("Processing user request",
        "user_id", userID,
        "operation", "get_profile",
        "timestamp", time.Now())
    
    // Error logging
    if err := processUser(ctx, userID); err != nil {
        logger.Error("Failed to process user",
            log.ErrorAttr(err),
            "user_id", userID,
            "retry_count", 3)
        return
    }
    
    logger.Info("User request completed successfully",
        "user_id", userID,
        "duration_ms", 150)
}
```

### Log Levels

```go
logger := log.FromContext(ctx)

// Different log levels
logger.Debug("Debug information", "details", debugInfo)
logger.Info("General information", "event", "user_login")
logger.Warn("Warning condition", "threshold_exceeded", true)
logger.Error("Error occurred", log.ErrorAttr(err))
```

### Service-Level Log Configuration

```go
// Configure logging at service level
service, err := patron.New("my-service", "1.0.0",
    // Enable JSON logging for production
    patron.WithJSONLogger(),
    
    // Add global log fields
    patron.WithLogFields(
        slog.String("environment", "production"),
        slog.String("service_type", "api"),
        slog.String("region", "us-east-1"),
        slog.Int("version", 2),
    ),
)
```

### Environment-Based Configuration

```bash
# Set log level via environment variable
export PATRON_LOG_LEVEL=debug  # debug, info, warn, error
```

## Distributed Tracing

### Automatic Tracing

Patron automatically creates traces for:
- HTTP requests and responses
- gRPC calls
- Database operations (Redis, MongoDB, etc.)
- Message queue operations (Kafka, AMQP, SQS)

### Custom Spans

```go
import (
    "github.com/beatlabs/patron/observability/trace"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)

func businessOperation(ctx context.Context, userID string) error {
    // Create custom span
    ctx, span := trace.StartSpan(ctx, "business-operation")
    defer span.End()
    
    // Add attributes
    span.SetAttributes(
        attribute.String("user.id", userID),
        attribute.String("operation.type", "profile_update"),
        attribute.Int("operation.version", 2),
    )
    
    // Perform operation
    err := updateUserProfile(ctx, userID)
    if err != nil {
        // Record error
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    
    // Mark as successful
    span.SetStatus(codes.Ok, "operation completed")
    return nil
}
```

### Nested Spans

```go
func complexOperation(ctx context.Context) error {
    ctx, span := trace.StartSpan(ctx, "complex-operation")
    defer span.End()
    
    // Step 1
    if err := step1(ctx); err != nil {
        span.RecordError(err)
        return err
    }
    
    // Step 2
    if err := step2(ctx); err != nil {
        span.RecordError(err)
        return err
    }
    
    return nil
}

func step1(ctx context.Context) error {
    ctx, span := trace.StartSpan(ctx, "step-1")
    defer span.End()
    
    span.SetAttributes(attribute.String("step", "validation"))
    
    // Validation logic
    return validateInput(ctx)
}

func step2(ctx context.Context) error {
    ctx, span := trace.StartSpan(ctx, "step-2")
    defer span.End()
    
    span.SetAttributes(attribute.String("step", "processing"))
    
    // Processing logic
    return processData(ctx)
}
```

### Trace Propagation

Traces automatically propagate across:

```go
// HTTP client calls
resp, err := http.Get("https://api.example.com/users")

// Database operations
user, err := userRepo.GetByID(ctx, userID)

// Message publishing
err = publisher.Publish(ctx, message)
```

## Metrics Collection

### Built-in Metrics

Patron automatically collects metrics for:
- HTTP request duration and count
- gRPC call duration and count
- Database operation duration and count
- Message processing duration and count

### Custom Metrics

```go
import (
    patronmetric "github.com/beatlabs/patron/observability/metric"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

var (
    // Counter for tracking events
    userLoginCounter = patronmetric.Int64Counter(
        "user_logins_total",
        "Total number of user logins",
    )
    
    // Histogram for tracking durations
    operationDuration = patronmetric.Int64Histogram(
        "operation_duration",
        "Operation duration in milliseconds",
        "ms",
    )
    
    // Gauge for tracking current values
    activeConnections = patronmetric.Int64UpDownCounter(
        "active_connections",
        "Number of active connections",
    )
)

func recordMetrics(ctx context.Context, userID string, duration time.Duration) {
    // Common attributes
    attrs := []attribute.KeyValue{
        attribute.String("user_type", "premium"),
        attribute.String("region", "us-east-1"),
    }
    
    // Increment counter
    userLoginCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
    
    // Record duration
    operationDuration.Record(ctx, duration.Milliseconds(), 
        metric.WithAttributes(attrs...))
    
    // Update gauge
    activeConnections.Add(ctx, 1, metric.WithAttributes(attrs...))
}
```

### Business Metrics

```go
var (
    ordersProcessed = patronmetric.Int64Counter(
        "orders_processed_total",
        "Total number of orders processed",
    )
    
    orderValue = patronmetric.Float64Histogram(
        "order_value",
        "Order value in USD",
        "USD",
    )
    
    inventoryLevel = patronmetric.Int64ObservableGauge(
        "inventory_level",
        "Current inventory level",
    )
)

func processOrder(ctx context.Context, order Order) error {
    start := time.Now()
    
    // Process order logic
    err := processOrderLogic(ctx, order)
    
    // Record metrics
    attrs := []attribute.KeyValue{
        attribute.String("product_category", order.Category),
        attribute.String("payment_method", order.PaymentMethod),
        attribute.String("status", getOrderStatus(err)),
    }
    
    ordersProcessed.Add(ctx, 1, metric.WithAttributes(attrs...))
    orderValue.Record(ctx, order.Value, metric.WithAttributes(attrs...))
    
    return err
}
```

## Health Checks

### Built-in Endpoints

Patron automatically provides health check endpoints:

- **`/alive`**: Liveness probe - returns 200 if service is running
- **`/ready`**: Readiness probe - returns 200 if service is ready
- **`/metrics`**: Prometheus metrics endpoint

### Custom Health Checks

```go
import (
    "github.com/beatlabs/patron/component/http/check"
)

func createHealthChecks() []check.Check {
    return []check.Check{
        // Database connectivity check
        check.New("database", func(ctx context.Context) error {
            return db.Ping(ctx)
        }),
        
        // External service check
        check.New("external-api", func(ctx context.Context) error {
            resp, err := http.Get("https://api.example.com/health")
            if err != nil {
                return err
            }
            defer resp.Body.Close()
            
            if resp.StatusCode != http.StatusOK {
                return fmt.Errorf("external API unhealthy: %d", resp.StatusCode)
            }
            return nil
        }),
        
        // Cache connectivity check
        check.New("cache", func(ctx context.Context) error {
            return redisClient.Ping(ctx).Err()
        }),
    }
}

// Add to HTTP component
func createHTTPComponent() (patron.Component, error) {
    rt, err := router.New(
        router.WithHealthChecks(createHealthChecks()...),
    )
    if err != nil {
        return nil, err
    }
    
    return patronhttp.New(rt)
}
```

## Configuration

### OpenTelemetry Configuration

```bash
# Service identification
export OTEL_SERVICE_NAME=my-service
export OTEL_SERVICE_VERSION=1.0.0

# Tracing configuration
export OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:14268/api/traces
export OTEL_TRACES_SAMPLER=traceidratio
export OTEL_TRACES_SAMPLER_ARG=0.1  # Sample 10% of traces

# Metrics configuration
export OTEL_METRICS_EXPORTER=prometheus
export OTEL_EXPORTER_PROMETHEUS_PORT=9090

# Resource attributes
export OTEL_RESOURCE_ATTRIBUTES=environment=production,region=us-east-1
```

### Patron-Specific Configuration

```bash
# Logging configuration
export PATRON_LOG_LEVEL=info
export PATRON_HTTP_DEFAULT_PORT=8080
```

## Best Practices

### Logging Best Practices

1. **Use structured logging** with consistent field names
2. **Include trace correlation** by using `log.FromContext(ctx)`
3. **Log at appropriate levels** - avoid debug logs in production
4. **Include relevant context** like user IDs, request IDs, etc.
5. **Don't log sensitive information** like passwords or tokens

```go
// Good
logger.Info("User authenticated",
    "user_id", userID,
    "method", "oauth",
    "duration_ms", authDuration.Milliseconds())

// Bad
logger.Info(fmt.Sprintf("User %s authenticated via %s", userID, method))
```

### Tracing Best Practices

1. **Create spans for significant operations**
2. **Add meaningful attributes** to spans
3. **Record errors** when they occur
4. **Set appropriate span status**
5. **Keep span names consistent** and descriptive

### Metrics Best Practices

1. **Use appropriate metric types** (counter, histogram, gauge)
2. **Include relevant labels** but avoid high cardinality
3. **Use consistent naming conventions**
4. **Document metric meanings**
5. **Monitor metric cardinality** to avoid performance issues

```go
// Good - low cardinality labels
userLoginCounter.Add(ctx, 1, metric.WithAttributes(
    attribute.String("method", "oauth"),
    attribute.String("region", "us-east-1"),
))

// Bad - high cardinality labels
userLoginCounter.Add(ctx, 1, metric.WithAttributes(
    attribute.String("user_id", userID), // Too many unique values
    attribute.String("timestamp", time.Now().String()),
))
```
