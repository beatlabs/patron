# AGENTS

## Build, test, and development workflows

- Build: `make` (default runs tests). Format: `make fmt` (go fmt). Format check: `make fmtcheck` (script/gofmtcheck.sh). Lint: `make lint` (golangci-lint with vendor mode); deep lint: `make deeplint`.
- Test all: `make test` → `go test ./... -cover -race -timeout 60s`. Integration tests: `make testint` (or `make testint-nocache`). CI runs with `-tags=integration` excluding `examples` and `encoding/protobuf/test`.
- Run single test: replace with your package/path and test name:
  - By name: `go test ./path/to/pkg -run ^TestName$ -v -race`.
  - By file: `go test ./path/to/pkg -run ^TestName$ -v -race ./path/to/pkg/file_test.go` (Go filters by package; prefer -run). With integration tags: add `-tags=integration`.
- Useful env/deps: start external deps for integrations via `make deps-start` (docker compose); stop via `make deps-stop`. Example apps: `make example-service`, `make example-client` (OTEL_EXPORTER_OTLP_INSECURE=true).
- CI: `.github/workflows/ci.yml` runs lint, format check, tests with integration tags, and e2e example tests. Codecov integration enabled.

## Lint configuration

- Lint config (.golangci.yml): enables errcheck, errorlint, revive, staticcheck, govet, gosec, sqlclosecheck, rowserrcheck, exhaustive, prealloc, whitespace, sloglint, spancheck, testifylint, and more. Build tag `integration` enabled; vendor mode on; tests linted. Formatters: gofmt, gofumpt, goimports.

## Copilot instructions

- Copilot rules (.github/copilot-instructions.md): default_language go; test framework testify; use context.Context first param for blocking/request-scoped; wrap errors with fmt.Errorf or errors.Join; follow Go idioms; avoid globals (except const/config); document exported symbols; use structured logging with slog.

## Code style and conventions

- Imports: standard → third-party → module-local; managed by goimports/gofumpt. No unused imports. Keep vendor modules pinned.
- Formatting: enforce gofmt; CI fails on deviations (use `make fmt`); prefer gofumpt-compatible style. Keep lines simple; avoid long one-liners.
- Types and naming: exported identifiers use PascalCase with doc comments; unexported use lowerCamelCase; interfaces named with -er when it makes sense; avoid stutter. Use context.Context as first param when calls may block or are request-scoped.
- Errors: return sentinel or wrapped errors; never panic in library code; prefer `%w` with fmt.Errorf; use errors.Join when aggregating; check and handle errors (errcheck). Avoid swallowing errors; use errorlint patterns; compare errors with errors.Is/As.
- Logging/observability: use slog via observability/log; include context-aware logging; use OpenTelemetry for tracing/metrics; avoid global mutable state.
- Concurrency: prefer contexts, timeouts; avoid goroutine leaks; close resources (sqlclosecheck/rowserrcheck enforced). Guard shared state; avoid data races (tests run with -race).
- Testing: use testify (assert/require); mark integration tests with `//go:build integration`; use goleak for goroutine leak detection in TestMain.

## Architecture and framework structure

- Entry point: `Service` type in `service.go` orchestrates lifecycle, observability setup, and component execution.
- Components: implement `Run(context.Context) error`; service runs them in goroutines and aggregates errors with `errors.Join`.
- Default HTTP component: automatically started on service creation; exposes `/debug`, `/alive`, `/ready`, `/metrics` endpoints (see `component/http/check.go`).
- Routes: declared using `"GET /path"` pattern with `http.HandlerFunc`; created via `http.NewRoute(path, handler, options...)`.
- Middleware: type `middleware.Func func(next http.Handler) http.Handler`; applied via route options or router-level config.
- Router: `component/http/router` package provides router creation with `New(options...)` returning `*http.ServeMux`; supports custom alive/ready checks, profiling, middleware, routes.
- Service instantiation: functional options pattern via `patron.New(name, version, options...)` (since v0.74.0); see `options.go` for available options.

## Components and clients

- Async components: Kafka (`component/kafka`), RabbitMQ (`component/amqp`), AWS SQS (`component/sqs`).
- Sync components: HTTP (`component/http`), gRPC (`component/grpc`).
- Clients: HTTP (`client/http`), gRPC (`client/grpc`), Kafka (`client/kafka`), RabbitMQ (`client/amqp`), AWS SQS/SNS (`client/sqs`, `client/sns`), Redis (`client/redis`), MongoDB (`client/mongo`), SQL (`client/sql`), Elasticsearch (`client/es`), MQTT (`client/mqtt`).

## Observability setup

- Observability: setup via `observability.Setup` in `Service.New`; shutdown in `Service.Run` defer.
- Logging: use `observability/log` for slog helpers; default attrs include service/version/host.
- Tracing: use `observability/trace` helpers and OTel SDK; create resource with semconv attributes.
- Metrics: use `observability/metric` helpers for OTel metrics.
- Semconv: currently using `go.opentelemetry.io/otel/semconv/v1.37.0` (see `observability/observability.go`); update import path when OTel is upgraded.

## Packages and utilities

- Cache: `cache/` package provides `Cache` and `TTLCache` interfaces; implementations include LRU (`cache/lru`) and Redis (`cache/redis`). Context passed as first param (since v0.75.0).
- Reliability: circuit breaker (`reliability/circuitbreaker`) and retry (`reliability/retry`) patterns with OTel metrics integration.
- Correlation: correlation ID propagation via `correlation/` package; header `X-Correlation-Id`.
- Encoding: JSON and protobuf encoders in `encoding/` package.

## Environment variables

- `PATRON_HTTP_DEFAULT_PORT`: override default HTTP port (default: 50000).
- `PATRON_HTTP_READ_TIMEOUT`: HTTP read timeout.
- `PATRON_HTTP_WRITE_TIMEOUT`: HTTP write timeout.
- `PATRON_HTTP_STATUS_ERROR_LOGGING`: enable/disable HTTP error logging.
- `PATRON_LOG_LEVEL`: set log level (debug, info, warn, error).
- `OTEL_EXPORTER_OTLP_INSECURE`: set to "true" for insecure OTLP exporter (used in examples/tests).

## Contribution workflow

- Issues: use `.github/ISSUE_TEMPLATE.md`; describe problem and solution.
- PRs: use `.github/PULL_REQUEST_TEMPLATE.md`; reference issue; sign commits (see `SIGNYOURWORK.md`); ensure tests pass; high coverage required.
- Breaking changes: document in `BREAKING.md` with migration guide.
- Keep changes formatted and lint-clean; add tests with testify; follow `docs/ContributionGuidelines.md`; CI must pass.

## Upgrade workflow

- See `.github/prompts/upgrade.prompt.md` for automated upgrade flow: create branch, upgrade modules/Docker images, test, commit, create PR, merge after CI success.
- After OTel upgrades: update `semconv` imports to matching version; run `go mod tidy && go mod vendor`.
