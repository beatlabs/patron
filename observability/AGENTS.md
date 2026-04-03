# observability

OpenTelemetry setup + slog logging. Central initialization for all components/clients.

## Structure

```
observability/
├── observability.go   # Setup/Shutdown, Provider, Config, attribute helpers
├── log/log.go         # slog setup, context-aware logging, level management
├── trace/tracing.go   # OTel TracerProvider helpers
└── metric/meter.go    # OTel MeterProvider helpers
```

## Code map

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `Config` | struct | `observability.go:55` | Name, Version, LogConfig for setup |
| `Setup` | func | `observability.go:64` | Initializes tracing + metrics providers |
| `Provider` | struct | `observability.go:24` | Holds MeterProvider + TracerProvider; `Shutdown()` flushes |
| `ComponentAttribute` | func | `observability.go:37` | Creates `attribute.String("component", name)` |
| `ClientAttribute` | func | `observability.go:42` | Creates `attribute.String("client", name)` |
| `StatusAttribute` | func | `observability.go:47` | Creates status attribute from string |
| `SucceededAttribute` | var | `observability.go:31` | Pre-built "succeeded" status attr |
| `FailedAttribute` | var | `observability.go:33` | Pre-built "failed" status attr |
| `log.Setup` | func | `log/log.go:22` | Configures slog with attrs, JSON mode, level |
| `log.FromContext` | func | `log/log.go:66` | Extracts logger from context |
| `log.WithContext` | func | `log/log.go:77` | Injects logger into context |

## Conventions

- `Setup` called once in `Service.New`; `Provider.Shutdown` deferred in `Service.Run`.
- Semconv: currently `go.opentelemetry.io/otel/semconv/v1.39.0` — update import path on OTel upgrades.
- Logging: always context-aware via `log.FromContext(ctx)`. Default attrs: service name, version, host.
- Components use `ComponentAttribute`; clients use `ClientAttribute` for span/metric attribution.
- Log level configurable via `PATRON_LOG_LEVEL` env var or `log.SetLevel`.
