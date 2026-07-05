## Purpose

Define the accepted Kafka client option behavior for Patron's franz-go based producer.

## Requirements

### Requirement: Kafka client uses Patron functional options

The Kafka client constructor SHALL expose Patron-owned functional options for optional configuration instead of requiring callers to pass franz-go option values directly.

#### Scenario: Constructing with no optional settings

- **WHEN** a caller creates a Kafka producer with broker addresses and no optional settings
- **THEN** the producer is created with the existing default broker, telemetry hook, and logger configuration

#### Scenario: Constructing with Patron options

- **WHEN** a caller creates a Kafka producer with broker addresses and Kafka client functional options
- **THEN** the constructor applies those options before creating the underlying franz-go client

### Requirement: Kafka client preserves franz-go option passthrough

The Kafka client SHALL provide a functional option that accepts one or more raw franz-go `kgo.Opt` values and includes them in the underlying franz-go client configuration.

#### Scenario: Passing raw franz-go options

- **WHEN** a caller supplies raw franz-go options through the passthrough functional option
- **THEN** those options are included in the final franz-go client configuration

#### Scenario: Passing multiple raw franz-go options

- **WHEN** a caller supplies multiple raw franz-go options through the passthrough functional option
- **THEN** all supplied options are preserved in order relative to each other

### Requirement: Kafka option migration is documented

The project SHALL document how callers migrate from direct `kgo.Opt` constructor arguments to the Patron functional option form.

#### Scenario: Reading Kafka client API documentation

- **WHEN** a maintainer or caller reads the Kafka client API documentation
- **THEN** the documentation shows the Patron option form and the franz-go passthrough form

#### Scenario: Breaking constructor change

- **WHEN** the implementation changes the public Kafka constructor signature incompatibly
- **THEN** `BREAKING.md` documents the old usage, new usage, and migration path
