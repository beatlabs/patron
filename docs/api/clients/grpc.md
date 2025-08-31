# gRPC client

Patron offers a thin gRPC client helper that wires OpenTelemetry tracing/metrics.

- Package: `github.com/beatlabs/patron/client/grpc`
- Function: `NewClient(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)`
- Defaults: adds `otelgrpc` stats handler to the dial options you provide

## Quick start

```go
import (
    patrongrpc "github.com/beatlabs/patron/client/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

// Create a client connection to a target (host:port or scheme-based)
cc, err := patrongrpc.NewClient("localhost:50051",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
if err != nil { /* handle */ }

defer cc.Close()

client := examples.NewGreeterClient(cc)
reply, err := client.SayHello(ctx, &examples.HelloRequest{FirstName: "John", LastName: "Doe"})
```

A runnable example client is in `examples/client/main.go` (use `-modes=grpc`).

## Notes

- If you pass no dial options, Patron creates an empty slice and then appends the OTel stats handler. You still need to provide transport credentials or other options as required by your environment.
- Tracing export is handled by your process' OpenTelemetry setup (see `observability/`).
