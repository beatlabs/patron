## Context

`client/kafka.New` currently accepts `opts ...kgo.Opt` directly. That keeps full franz-go configurability, but it makes Kafka the only Patron client whose constructor does not use package-owned functional options.

The change should align Kafka with the `client/` convention without hiding franz-go's broad option surface. The safest implementation is a small option layer that stores raw `kgo.Opt` values internally and exposes a passthrough for advanced callers.

## Goals / Non-Goals

**Goals:**

- Make the Kafka client constructor use Patron-owned functional options.
- Keep access to arbitrary franz-go options through a single passthrough option.
- Preserve current built-in defaults: seed brokers, OTel hooks, slog logger, validation, producer behavior, metrics, tracing, and correlation propagation.
- Document how callers migrate from direct `kgo.Opt` arguments.

**Non-Goals:**

- Wrap every franz-go option.
- Replace franz-go or change producer send semantics.
- Add new runtime dependencies.
- Redesign the broader client package option conventions.

## Decisions

1. Add `client/kafka.OptionFunc`.

   `OptionFunc` should mutate an unexported config struct owned by `client/kafka`. The initial config can hold `kgoOpts []kgo.Opt`; future Kafka-specific options can be added without changing the constructor shape.

   Alternative considered: keep `opts ...kgo.Opt` and only document the exception. That avoids API churn but leaves the inconsistency from issue #1053 unresolved.

2. Add `WithKafkaOptions(opts ...kgo.Opt) OptionFunc`.

   This keeps franz-go's complete option surface available without creating Patron wrappers for 100+ upstream options. It also gives callers a direct migration path from `New(brokers, kgo.ClientID("x"))` to `New(brokers, WithKafkaOptions(kgo.ClientID("x")))`.

   Alternative considered: add wrappers for common franz-go options only. That creates an arbitrary maintenance surface and still needs passthrough for less common options.

3. Keep Patron defaults prepended before caller options.

   The constructor should continue appending seed brokers, OTel hooks, and the slog logger before caller-provided options, matching current behavior. If franz-go treats later options as overriding earlier ones, callers retain the same ability they have today to override compatible settings.

4. Treat the constructor signature change as a breaking API change unless compatibility is deliberately added.

   Go cannot overload `New`. A direct switch from `opts ...kgo.Opt` to `opts ...OptionFunc` will break callers passing raw franz-go options. If the project wants source compatibility, add a separate helper such as `NewWithKafkaOptions` or defer the constructor change. Otherwise document the migration in `BREAKING.md`.

## Risks / Trade-offs

- Breaking downstream callers -> document the migration and consider a compatibility helper if maintainers prefer a softer transition.
- Option ordering changes -> preserve the existing append order and add tests that inspect observable option effects where practical.
- Over-wrapping franz-go -> keep only `WithKafkaOptions` initially unless a Patron-specific option has clear value.
- Stale Kafka docs -> update `docs/api/clients/kafka.md`, which currently does not match the franz-go API shape.

## Migration Plan

1. Introduce `OptionFunc`, internal config, and `WithKafkaOptions`.
2. Update `New` to accept `opts ...OptionFunc` and convert config into the final `[]kgo.Opt`.
3. Update tests for validation, default construction, and passthrough options.
4. Update Kafka docs, client guidance, and `BREAKING.md` if the constructor signature changes.
5. Run `make test`, `make lint`, and `make fmtcheck`; run integration tests if Kafka-backed behavior is changed beyond constructor option wiring.

## Open Questions

- Should implementation preserve source compatibility with an additional helper, or is a documented breaking change acceptable?
- Should any Patron-specific Kafka options be added now, or should the first change expose only `WithKafkaOptions`?
