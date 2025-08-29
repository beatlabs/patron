# AGENTS

- Build: `make` (default runs tests). Format: `make fmt` (go fmt). Format check: `make fmtcheck` (script/gofmtcheck.sh). Lint: `make lint` (golangci-lint with vendor mode); deep lint: `make deeplint`.
- Test all: `make test` → `go test ./... -cover -race -timeout 60s`. Integration tests: `make testint` (or `make testint-nocache`). CI runs with `-tags=integration` excluding `examples` and `encoding/protobuf/test`.
- Run single test: replace with your package/path and test name:
  - By name: `go test ./path/to/pkg -run ^TestName$ -v -race`.
  - By file: `go test ./path/to/pkg -run ^TestName$ -v -race ./path/to/pkg/file_test.go` (Go filters by package; prefer -run). With integration tags: add `-tags=integration`.
- Useful env/deps: start external deps for integrations via `make deps-start` (docker compose); stop via `make deps-stop`. Example apps: `make example-service`, `make example-client` (OTEL_EXPORTER_OTLP_INSECURE=true).
- Lint config highlights (.golangci.yml): enables errcheck, errorlint, revive, staticcheck, govet, gosec, sqlclosecheck, rowserrcheck, exhaustive, prealloc, whitespace, gofmt/gofumpt/goimports formatters. Build tag `integration` enabled; vendor mode on; tests linted.
- Copilot rules (.github/copilot-instructions.md): default_language go; test framework testify; use context.Context first param for blocking/request-scoped; wrap errors with fmt.Errorf or errors.Join; follow Go idioms; avoid globals (except const/config); document exported symbols; use structured logging with slog.
- Code style:
  - Imports: standard → third-party → module-local; managed by goimports/gofumpt. No unused imports. Keep vendor modules pinned.
  - Formatting: enforce gofmt; CI fails on deviations (use `make fmt`); prefer gofumpt-compatible style. Keep lines simple; avoid long one-liners.
  - Types and naming: exported identifiers use PascalCase with doc comments; unexported use lowerCamelCase; interfaces named with -er when it makes sense; avoid stutter. Use context.Context as first param when calls may block or are request-scoped.
  - Errors: return sentinel or wrapped errors; never panic in library code; prefer `%w` with fmt.Errorf; use errors.Join when aggregating; check and handle errors (errcheck). Avoid swallowing errors; use errorlint patterns; compare errors with errors.Is/As.
  - Logging/observability: use slog via observability/log; include context-aware logging; use OpenTelemetry for tracing/metrics; avoid global mutable state.
  - Concurrency: prefer contexts, timeouts; avoid goroutine leaks; close resources (sqlclosecheck/rowserrcheck enforced). Guard shared state; avoid data races (tests run with -race).
- Commit/PR: keep changes formatted and lint-clean; add tests with testify; follow ContributionGuidelines.md; CI must pass.
