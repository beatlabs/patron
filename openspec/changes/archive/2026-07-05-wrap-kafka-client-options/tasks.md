## 1. API Decision

- [x] 1.1 Decide whether `client/kafka.New` will be a documented breaking change or whether a compatibility helper will be added.
- [x] 1.2 Run GitNexus impact analysis for `client/kafka.New` before editing the constructor.

## 2. Kafka Option Implementation

- [x] 2.1 Add `client/kafka.OptionFunc` and an internal Kafka client option config.
- [x] 2.2 Add `WithKafkaOptions(...kgo.Opt) OptionFunc` to preserve raw franz-go option passthrough.
- [x] 2.3 Update `client/kafka.New` to apply Patron options and build the final franz-go option list with existing defaults preserved.
- [x] 2.4 Keep broker validation, OTel hooks, slog logger setup, producer send behavior, metrics, tracing, and correlation behavior unchanged.

## 3. Tests

- [x] 3.1 Update Kafka client unit tests for the new functional option constructor form.
- [x] 3.2 Add test coverage for `WithKafkaOptions` passthrough behavior.
- [x] 3.3 Add or update tests covering invalid option handling if the option application path can return errors.

## 4. Documentation

- [x] 4.1 Update `docs/api/clients/kafka.md` to show the Patron option form and franz-go passthrough usage.
- [x] 4.2 Update `client/AGENTS.md` to remove or clarify the Kafka option exception.
- [x] 4.3 Update `BREAKING.md` with old usage, new usage, and migration path if the constructor signature changes incompatibly.

## 5. Verification

- [x] 5.1 Run `make fmtcheck`.
- [x] 5.2 Run `make test`.
- [x] 5.3 Run `make lint`.
- [x] 5.4 Run `make deps-start`, `make testint`, and `make deps-stop` if the implementation changes Kafka-backed behavior beyond constructor option wiring. Not required; implementation is constructor option wiring only.
- [x] 5.5 Run GitNexus change detection before committing.
