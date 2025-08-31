# gRPC component

This component runs a gRPC server with sane defaults and built‑in observability (OpenTelemetry tracing/metrics) and health.

- Package: `github.com/beatlabs/patron/component/grpc`
- Component type: implements `Run(ctx context.Context) error`
- Defaults: registers gRPC health server, adds OTel stats handler, optional server reflection

## Quick start

```go
import (
    patrongrpc "github.com/beatlabs/patron/component/grpc"
)

// Create component and register your service implementations.
cmp, err := patrongrpc.New(50051)
if err != nil { /* handle */ }

// Register generated gRPC servers on cmp.Server().
// examples.RegisterGreeterServer is generated from greeter.proto.
examples.RegisterGreeterServer(cmp.Server(), greeterImpl)

// Add the component to your Patron service and run.
svc, _ := patron.New("example-service", patron.WithComponents(cmp))
_ = svc.Run(ctx)
```

See a full runnable example in `examples/service/grpc.go`.

## Options

```go
// WithServerOptions: provide raw grpc.ServerOption values
cmp, err := patrongrpc.New(50051,
    patrongrpc.WithServerOptions(
        // e.g., add interceptors, keepalive, limits, creds, etc.
        grpc.UnaryInterceptor(myUnaryInterceptor),
    ),
)

// WithReflection: opt‑in to server reflection (useful in local/dev tools)
cmp, err := patrongrpc.New(50051, patrongrpc.WithReflection())
```

Notes

- Server options you pass replace any previously set options on the component. Patron always appends an OTel `StatsHandler` internally for tracing/metrics.
- Reflection can be a security risk when exposed publicly. Prefer enabling only in non‑prod or behind auth.

## Observability

- Tracing and metrics are automatically wired via `otelgrpc` stats handlers. Ensure you call `observability.Setup` (done by `Service.Run`) or the helpers in `observability/` to export to your backend.
- The component logs a startup line with the port and gracefully stops when the service context is canceled.

## Health

A standard gRPC health service is registered by default (`grpc.health.v1.Health`). You can wire your own health reporting using that API if needed.

## Try the example

- Start the example service (includes HTTP, Kafka, etc., but also gRPC):

```sh
make example-service
```

- From the example client, run only the gRPC path:

```sh
make example-client ARGS='-modes=grpc'
```

The client calls `SayHello` on the example Greeter service defined in `examples/greeter.proto`.
