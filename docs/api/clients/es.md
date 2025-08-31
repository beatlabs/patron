# Elasticsearch client

OpenTelemetry-instrumented Elasticsearch client with request duration metrics.

- Package: `github.com/beatlabs/patron/client/es`
- Upstream: `github.com/elastic/go-elasticsearch/v8`

## Usage

```go
import (
  esclient "github.com/beatlabs/patron/client/es"
  "github.com/elastic/go-elasticsearch/v8"
)

cfg := elasticsearch.Config{ /* hosts, auth, transport, etc. */ }
cl, err := esclient.New(cfg, "v1")
```

- Tracing is provided by upstream OTEL instrumentation.
- Patron adds an OTEL histogram metric `es.request.duration` labeled with endpoint and status.
