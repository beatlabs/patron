## Why

The Kafka client is the only client package that exposes its underlying library option type directly, accepting `kgo.Opt` values instead of Patron-style functional options. This creates an API consistency exception in `client/` and makes the Kafka client diverge from the constructor pattern documented for Patron clients.

## What Changes

- Add a Patron `OptionFunc` option layer for `client/kafka`.
- Preserve access to franz-go options through a passthrough option such as `WithKafkaOptions(...kgo.Opt)`.
- Update the Kafka client constructor to accept Patron options while keeping built-in instrumentation, broker validation, logging, and correlation behavior.
- Add tests covering option application, passthrough franz-go options, and existing constructor validation.
- Update Kafka client docs and client guidance to describe the Kafka option pattern and migration path.

## Capabilities

### New Capabilities

- `kafka-client-options`: Kafka producers can be configured through Patron functional options while still allowing callers to pass raw franz-go options when needed.

### Changed Capabilities

- None.

## Impact

- Affected code: `client/kafka/`, `docs/api/clients/kafka.md`, and `client/AGENTS.md`.
- Public API: Kafka client constructor and option surface.
- Compatibility: Potentially breaking if `New` no longer accepts `opts ...kgo.Opt` directly; design should choose whether to preserve a compatibility path or make the migration explicit.
- Dependencies: No new dependencies expected; continue using franz-go `kgo.Opt` internally.
- Related issue: https://github.com/beatlabs/patron/issues/1053.
