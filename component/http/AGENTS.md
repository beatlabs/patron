# component/http

HTTP server component. Routes + middleware + caching + auth + router.

## Structure

```
http/
├── component.go, component_option.go  # Component lifecycle + options (TLS, timeouts, port)
├── route.go, route_option.go          # Route definition + per-route options
├── check.go                           # /alive, /ready, /metrics, /debug endpoints
├── observability.go                   # Request tracing + metrics integration
├── middleware/                         # middleware.Func + standard middlewares
│   ├── middleware.go                  # Recovery, auth, compression, rate limiting, caching, logging
│   └── logging.go                    # Request/response logging helpers
├── router/                            # Router construction
│   ├── router.go                     # New(options...) → *http.ServeMux
│   ├── router_option.go             # Router-level options (alive/ready checks, profiling, middleware)
│   └── route.go                     # Route registration helpers
├── cache/                             # HTTP response caching layer
│   ├── cache.go                      # Cache handler wrapping (342 LOC)
│   ├── model.go                     # Cache entry model
│   ├── route.go                     # Cached route creation
│   └── metric.go                    # Cache hit/miss metrics
└── auth/                              # Auth middleware
    ├── auth.go                       # Authenticator interface
    └── apikey/apikey.go             # API key authenticator implementation
```

## Code map

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `Component` | struct | `component.go:69` | HTTP server with `Run(ctx)` lifecycle |
| `New` | func | `component.go:83` | `New(handler, opts...)` — creates HTTP component |
| `OptionFunc` | func | `component_option.go:9` | `WithTLS`, `WithPort`, `WithReadTimeout`, etc. |
| `Route` | struct | `route.go:16` | Path + handler + middlewares |
| `NewRoute` | func | `route.go:40` | `NewRoute(path, handler, opts...)` |
| `Routes` | struct | `route.go:66` | Accumulator with `Append`/`Result` for error collection |
| `Func` | type | `middleware/middleware.go:86` | `func(next http.Handler) http.Handler` |
| `Chain` | func | `middleware/middleware.go:470` | Chains multiple middleware.Func in order |
| `router.New` | func | `router/router.go:30` | `New(opts...) (*http.ServeMux, error)` |

## Conventions

- Routes: `"METHOD /path"` pattern (Go 1.22+ ServeMux), e.g. `"GET /api/users"`.
- Middleware order matters: applied left-to-right via `Chain` or route options.
- Default endpoints (`/alive`, `/ready`, `/metrics`, `/debug`) registered automatically by router.
- Component reads `PATRON_HTTP_DEFAULT_PORT` (default 50000), `PATRON_HTTP_READ_TIMEOUT`, `PATRON_HTTP_WRITE_TIMEOUT` from env.
- Cache layer wraps handlers; uses `cache.Cache` interface from `cache/` package.

## Available middlewares

`NewRecovery`, `NewAppNameVersion`, `NewAuth`, `NewLoggingTracing`, `NewInjectObservability`, `NewRateLimiting`, `NewCompression`, `NewCaching`.

## Anti-patterns

- **Never** bypass middleware chain — always use `Route` options or `router.Config`.
- **Never** write directly to `http.ResponseWriter` in middleware — use the wrapped `responseWriter`.
