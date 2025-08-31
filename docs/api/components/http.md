# HTTP component (server)

Serve HTTP endpoints with built-in profiling, health checks, middleware, and tracing/logging.

- Component: `github.com/beatlabs/patron/component/http`
- Router: `github.com/beatlabs/patron/component/http/router`
- Type: component implementing `Run(ctx context.Context) error`

## Quick start

```go
import (
  "context"
  "net/http"
  patronhttp "github.com/beatlabs/patron/component/http"
  httprouter "github.com/beatlabs/patron/component/http/router"
)

hello, _ := patronhttp.NewRoute("GET /api/hello", func(w http.ResponseWriter, r *http.Request) {
  w.WriteHeader(http.StatusOK)
  _, _ = w.Write([]byte("hello"))
})

mux, _ := httprouter.New(
  httprouter.WithRoutes(hello),
)

cmp, _ := patronhttp.New(mux, patronhttp.WithPort(8080))
_ = cmp.Run(context.Background()) // usually managed by Service
```

See `examples/service/` for a full setup.

## Component options

- `WithPort(port int)`
- `WithTLS(certFile, keyFile string)`
- `WithReadTimeout(d time.Duration)`
- `WithWriteTimeout(d time.Duration)`
- `WithHandlerTimeout(d time.Duration)`
- `WithShutdownGracePeriod(d time.Duration)`

Env overrides: `PATRON_HTTP_DEFAULT_PORT`, `PATRON_HTTP_READ_TIMEOUT`, `PATRON_HTTP_WRITE_TIMEOUT`.

## Router options

- `WithRoutes(routes...)`
- `WithAliveCheck(func() AliveStatus)` (defaults to Alive)
- `WithReadyCheck(func() ReadyStatus)` (defaults to Ready)
- `WithDeflateLevel(level int)` (compression)
- `WithMiddlewares(mm ...)`
- `WithExpVarProfiling()` (adds `/debug/vars`)
- `WithAppNameHeaders(name, version)` (adds X-App-Name/X-App-Version)

Management routes: `/debug/pprof/*`, `/alive`, `/ready`.

## Route helpers

- `WithMiddlewares(mm ...)`
- `WithRateLimiting(limit, burst)`
- `WithAuth(authenticator)`
- `WithCache(cache, httpcache.Age)` (GET routes)
- `router.NewFileServerRoute("GET /", "./public", "./public/index.html")`

## Observability

- Logging/tracing middleware is applied to user routes.
- Configure error-status logging via `PATRON_HTTP_STATUS_ERROR_LOGGING` (comma-separated status codes).
- Change log level at runtime: `POST /debug/log/{level}`.
