# MongoDB client

Connect helper with integrated OpenTelemetry tracing via the official driver’s monitor and added duration metrics.

- Package: `github.com/beatlabs/patron/client/mongo`
- Upstream: `go.mongodb.org/mongo-driver/mongo`

## Usage

```go
cl, err := mongoclient.Connect(ctx, options.Client().ApplyURI(uri))
```

- Tracing is enabled using `otelmongo` monitor.
- Patron records an OTEL histogram `mongo.duration` with command name and status.
