# AGENTS

**Generated:** 2026-04-02 | **Commit:** cea3379a2 | **Branch:** master

## Overview

Go microservice framework (`github.com/beatlabs/patron`, Go 1.24). Component-based architecture: `Service` orchestrates lifecycle + observability; `Component` implementations handle sync/async processing; `client/` wraps external services. 153 source files, ~21K LOC.

## Structure

```
patron/
├── service.go, options.go, doc.go    # Service entry point + functional options
├── component/                        # Server-side processing (see component/http/AGENTS.md)
│   ├── http/                         # HTTP server (routes, middleware, cache, auth, router)
│   ├── kafka/                        # Kafka consumer (franz-go)
│   ├── amqp/                         # RabbitMQ consumer (amqp091-go)
│   ├── grpc/                         # gRPC server
│   └── sqs/                          # AWS SQS consumer
├── client/                           # Client wrappers (see client/AGENTS.md)
│   ├── http/, grpc/, kafka/, amqp/   # Protocol clients
│   ├── sqs/, sns/                    # AWS clients (SDK v2)
│   ├── sql/, redis/, mongo/, es/     # Data store clients
│   └── mqtt/                         # MQTT publisher
├── observability/                    # OTel setup (see observability/AGENTS.md)
│   ├── log/                          # slog helpers
│   ├── trace/                        # Tracing helpers
│   └── metric/                       # Metrics helpers
├── cache/                            # Cache interfaces + impls (see cache/AGENTS.md)
├── reliability/                      # Circuit breaker + retry (see reliability/AGENTS.md)
├── correlation/                      # X-Correlation-Id propagation
├── encoding/                         # JSON + protobuf codecs
├── examples/                         # Service + client examples (e2e tested in CI)
├── internal/                         # Test helpers + validation utils
└── docs/                             # API docs per component/client
```

## Where to look

| Task | Location | Notes |
|------|----------|-------|
| Add new component | `component/<protocol>/` | Implement `Component` interface (`Run(ctx) error`) |
| Add new client | `client/<service>/` | Follow functional options pattern, add OTel instrumentation |
| HTTP routes/middleware | `component/http/` | See `component/http/AGENTS.md` |
| Observability changes | `observability/` | See `observability/AGENTS.md` |
| Cache implementations | `cache/` | See `cache/AGENTS.md` |
| Reliability patterns | `reliability/` | See `reliability/AGENTS.md` |
| Breaking changes | `BREAKING.md` | Document migration path |
| CI pipeline | `.github/workflows/ci.yml` | Lint → format check → integration tests → e2e |

## Code map

| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `Service` | struct | `service.go:31` | Orchestrates components + observability lifecycle |
| `Component` | interface | `service.go:25` | `Run(context.Context) error` — all components implement this |
| `New` | func | `service.go:41` | `patron.New(name, version, opts...)` — creates Service |
| `OptionFunc` | func | `options.go:9` | Functional option pattern for Service config |
| `WithSIGHUP` | func | `options.go:12` | Register SIGHUP handler |
| `WithLogFields` | func | `options.go:26` | Add default log fields |
| `WithJSONLogger` | func | `options.go:46` | Switch to JSON log output |

## Commands

```bash
task                  # Run tests (default)
task test             # go test ./... -cover -race -timeout 60s
task testint          # Integration tests (-tags=integration), needs `task deps-start`
task lint             # golangci-lint (requires v2.6.1 installed)
task deeplint         # Deep lint analysis
task fmt              # go fmt
task fmtcheck         # Check formatting (CI uses this)
task deps-start       # Docker Compose up (Kafka, RabbitMQ, MySQL, etc.)
task deps-stop        # Docker Compose down
task example-service  # Run example service
task example-client   # Run example client
task validate         # Validate Taskfile.yml
task --list           # List all available tasks
```

## Build, test, and development workflows

- Single test: `go test ./path/to/pkg -run ^TestName$ -v -race`. Integration: add `-tags=integration`.
- Test files: `*_test.go` (unit), `*_integration_test.go` (integration with `//go:build integration`).
- goleak in `TestMain` for goroutine leak detection; testify `assert`/`require` for assertions.
- CI: `.github/workflows/ci.yml` — validates Taskfile, lint, format check, integration tests (excludes `examples` and `encoding/protobuf/test`), e2e example tests. Codecov enabled.
- Prerequisites: golangci-lint v2.6.1. Install: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.6.1`.
- Git worktrees: `wt <branch-name>` for parallel development.

## Lint configuration

- `.golangci.yml`: errcheck, errorlint, revive, staticcheck, govet, gosec, sqlclosecheck, rowserrcheck, exhaustive, prealloc, whitespace, sloglint, spancheck, testifylint, and more. Build tag `integration`; vendor mode; tests linted. Formatters: gofmt, gofumpt, goimports.

## Code style and conventions

- **Imports**: standard → third-party → module-local (goimports manages).
- **Naming**: PascalCase exported (with doc comments), lowerCamelCase unexported, `-er` interfaces, no stutter.
- **Errors**: sentinel or wrapped (`%w`); `errors.Join` for aggregation; `errors.Is`/`As` for comparison; never panic in library code; never swallow errors.
- **Context**: `context.Context` as first param for blocking/request-scoped calls.
- **Concurrency**: contexts + timeouts; no goroutine leaks; close resources; guard shared state; `-race` enforced.
- **Functional options**: `OptionFunc` pattern for all constructors — `New(required, opts...)`.

## Anti-patterns (this project)

- **Never** panic in library code — return errors.
- **Never** use global mutable state — pass via context or constructor.
- **Never** swallow errors — errcheck linter enforced.
- **Never** skip `context.Context` as first param in blocking calls.
- **Never** leave goroutines unmanaged — goleak in TestMain catches leaks.
- **Never** use builder pattern — replaced by functional options in v0.74.0.

## Environment variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `PATRON_HTTP_DEFAULT_PORT` | HTTP server port | 50000 |
| `PATRON_HTTP_READ_TIMEOUT` | HTTP read timeout | — |
| `PATRON_HTTP_WRITE_TIMEOUT` | HTTP write timeout | — |
| `PATRON_HTTP_STATUS_ERROR_LOGGING` | Enable HTTP error logging | — |
| `PATRON_LOG_LEVEL` | Log level (debug/info/warn/error) | — |
| `OTEL_EXPORTER_OTLP_INSECURE` | Insecure OTLP exporter | false |

## Contribution workflow

- Issues: `.github/ISSUE_TEMPLATE.md`. PRs: `.github/PULL_REQUEST_TEMPLATE.md`.
- Sign commits (`SIGNYOURWORK.md`); high test coverage; lint-clean; CI must pass.
- Breaking changes: document in `BREAKING.md` with migration guide.
- OTel upgrades: update `semconv` import path to matching version; `go mod tidy && go mod vendor`.

## Subdirectory knowledge

| Path | AGENTS.md | Focus |
|------|-----------|-------|
| `component/http/` | ✅ | Routes, middleware, cache, auth, router |
| `client/` | ✅ | Client conventions, options, instrumentation |
| `observability/` | ✅ | OTel setup, logging, tracing, metrics |
| `cache/` | ✅ | Cache interfaces, LRU, Redis |
| `reliability/` | ✅ | Circuit breaker, retry |

# context-mode — MANDATORY routing rules

You have context-mode MCP tools available. These rules are NOT optional — they protect your context window from flooding. A single unrouted command can dump 56 KB into context and waste the entire session.

## BLOCKED commands — do NOT attempt these

### curl / wget — BLOCKED
Any shell command containing `curl` or `wget` will be intercepted and blocked by the context-mode plugin. Do NOT retry.
Instead use:
- `context-mode_ctx_fetch_and_index(url, source)` to fetch and index web pages
- `context-mode_ctx_execute(language: "javascript", code: "const r = await fetch(...)")` to run HTTP calls in sandbox

### Inline HTTP — BLOCKED
Any shell command containing `fetch('http`, `requests.get(`, `requests.post(`, `http.get(`, or `http.request(` will be intercepted and blocked. Do NOT retry with shell.
Instead use:
- `context-mode_ctx_execute(language, code)` to run HTTP calls in sandbox — only stdout enters context

### Direct web fetching — BLOCKED
Do NOT use any direct URL fetching tool. Use the sandbox equivalent.
Instead use:
- `context-mode_ctx_fetch_and_index(url, source)` then `context-mode_ctx_search(queries)` to query the indexed content

## REDIRECTED tools — use sandbox equivalents

### Shell (>20 lines output)
Shell is ONLY for: `git`, `mkdir`, `rm`, `mv`, `cd`, `ls`, `npm install`, `pip install`, and other short-output commands.
For everything else, use:
- `context-mode_ctx_batch_execute(commands, queries)` — run multiple commands + search in ONE call
- `context-mode_ctx_execute(language: "shell", code: "...")` — run in sandbox, only stdout enters context

### File reading (for analysis)
If you are reading a file to **edit** it → reading is correct (edit needs content in context).
If you are reading to **analyze, explore, or summarize** → use `context-mode_ctx_execute_file(path, language, code)` instead. Only your printed summary enters context.

### grep / search (large results)
Search results can flood context. Use `context-mode_ctx_execute(language: "shell", code: "grep ...")` to run searches in sandbox. Only your printed summary enters context.

## Tool selection hierarchy

1. **GATHER**: `context-mode_ctx_batch_execute(commands, queries)` — Primary tool. Runs all commands, auto-indexes output, returns search results. ONE call replaces 30+ individual calls.
2. **FOLLOW-UP**: `context-mode_ctx_search(queries: ["q1", "q2", ...])` — Query indexed content. Pass ALL questions as array in ONE call.
3. **PROCESSING**: `context-mode_ctx_execute(language, code)` | `context-mode_ctx_execute_file(path, language, code)` — Sandbox execution. Only stdout enters context.
4. **WEB**: `context-mode_ctx_fetch_and_index(url, source)` then `context-mode_ctx_search(queries)` — Fetch, chunk, index, query. Raw HTML never enters context.
5. **INDEX**: `context-mode_ctx_index(content, source)` — Store content in FTS5 knowledge base for later search.

## Output constraints

- Keep responses under 500 words.
- Write artifacts (code, configs, PRDs) to FILES — never return them as inline text. Return only: file path + 1-line description.
- When indexing content, use descriptive source labels so others can `search(source: "label")` later.

## ctx commands

| Command | Action |
|---------|--------|
| `ctx stats` | Call the `stats` MCP tool and display the full output verbatim |
| `ctx doctor` | Call the `doctor` MCP tool, run the returned shell command, display as checklist |
| `ctx upgrade` | Call the `upgrade` MCP tool, run the returned shell command, display as checklist |

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **patron** (3111 symbols, 6568 relationships, 267 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/patron/context` | Codebase overview, check index freshness |
| `gitnexus://repo/patron/clusters` | All functional areas |
| `gitnexus://repo/patron/processes` | All execution flows |
| `gitnexus://repo/patron/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
