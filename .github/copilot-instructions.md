# Copilot configuration for this repository
# For documentation, see: https://docs.github.com/en/copilot/configuration/copilot-configuration-in-your-repository

default_language: go

# We use the testify framework for testing. Copilot suggestions should not conflict with testify idioms.
test_framework: testify

# Code style and conventions:
- Use context.Context as the first argument in functions that may block or are request-scoped.
- Prefer error wrapping using fmt.Errorf or errors.Join when returning errors.
- Follow Go idioms for naming, error handling, and documentation.
- Avoid global variables except for constants or configuration.
- Document public functions and types with clear, concise comments.
- Use structured logging with slog.

---

## Guidance for AI coding agents

- Big picture
	- Patron is a Go microservice framework. Entry point is `Service` (`service.go`) which orchestrates Components (`Run(ctx) error`).
	- Default HTTP component exposes management endpoints: `/debug`, `/alive`, `/ready`, `/metrics` (see `component/http`). Routes are declared like `"GET /alive"` with plain handlers.
	- Observability (logs, traces, metrics) is set up via `observability.Setup` and shut down in `Service.Run` defer.

- Architecture patterns
	- Components implement `Run(context.Context) error`; service runs them in goroutines and aggregates errors via `errors.Join`.
	- Observability: use `observability/log` for slog; tracing via `observability/trace` helpers and OTel SDK. Create resource with semconv attributes.
	- Semconv import path currently: `go.opentelemetry.io/otel/semconv/v1.37.0` (see `observability/observability.go`). If OTel is bumped, align these paths.

- Developer workflows (Makefile)
	- `make` runs tests. `make fmt` to format, `make fmtcheck` to verify formatting, `make lint` for golangci-lint in Docker (vendor mode).
	- Unit tests: `make test` (`-race -cover`). Integration: `make testint` or `make testint-nocache` with `-tags=integration`.
	- Integration deps: `make deps-start` (docker compose up --wait), `make deps-stop` to tear down. Example apps: `make example-service`, `make example-client`.
    - `gh` cli interacts with GitHub. `gh pr create` to create PR, `gh pr status` to monitor the PR, `gh pr merge -s -d --repo beatlabs/patron` to merge(squash) on success.

- Conventions
	- Always pass `context.Context` first in blocking or request-scoped APIs.
	- Error handling: wrap with `%w`; use `errors.Is/As`; aggregate with `errors.Join`; avoid panics.
	- Logging: prefer `slog` with attrs from `observabilityConfig` (service/version/host).
	- Tests: use `testify` asserts/requires; mark external-deps tests with `//go:build integration`.
	- Go version: Go 1.24+; vendor mode is maintained and used by some tasks.

- Integration points
	- HTTP: `component/http` (routes, middleware, health checks in `check.go`).
	- gRPC: `component/grpc` with OTel interceptors.
	- Async: Kafka (`component/client/kafka` and `component/kafka`), RabbitMQ (`component/amqp`), AWS SQS (`component/sqs`).
	- Data: Redis (`client/redis` with `redisotel`), MongoDB (`client/mongo`), SQL/MySQL (`client/sql`).
	- Tracing/Metrics: configured in `observability/*`; use `patrontrace.StartSpan` and metric helpers.

- Upgrades
	- See `.github/prompts/upgrade.prompt.md` for the automated upgrade flow. After OTel upgrades, update `semconv` imports to the matching version and run `go mod tidy && go mod vendor`.

- Useful map
	- `service.go` (lifecycle), `component/*` (HTTP/GRPC/Kafka/etc.), `client/*` (client libs), `observability/*` (OTel + slog), `Makefile` (workflows), `docker-compose.yml` (integration deps).
