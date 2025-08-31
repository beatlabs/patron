# HTTP client

Traced HTTP client with correlation propagation, optional circuit breaker, and auto-decompression helpers.

- Package: `github.com/beatlabs/patron/client/http`
- Type: `TracedClient`, implements `Do(req *http.Request) (*http.Response, error)`

## Quick start

```go
import (
  clienthttp "github.com/beatlabs/patron/client/http"
)

cl, _ := clienthttp.New(
  clienthttp.WithTimeout(30*time.Second),
)

req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com/api", nil)
req.Header.Set("Accept-Encoding", "gzip")

rsp, err := cl.Do(req)
if err != nil { /* handle */ }

defer rsp.Body.Close()
```

Notes

- Tracing: uses `otelhttp.NewTransport` under the hood; spans are created for outbound requests.
- Correlation: adds the `X-Correlation-ID` header from request context automatically.
- Decompression: if `Accept-Encoding` is set to `gzip` or `deflate`, response body is decompressed.

## Options

- `WithTimeout(d time.Duration)`
- `WithCircuitBreaker(name string, set circuitbreaker.Setting)`
- `WithTransport(rt http.RoundTripper)` (wrapped with `otelhttp.NewTransport`)
- `WithCheckRedirect(func(req *http.Request, via []*http.Request) error)`

See tests in `client/http` and usage in examples for more.
