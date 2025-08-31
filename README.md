# patron ![CI](https://github.com/beatlabs/patron/workflows/CI/badge.svg) [![codecov](https://codecov.io/gh/beatlabs/patron/graph/badge.svg?token=sxY15rXW1X)](https://codecov.io/gh/beatlabs/patron) [![Go Report Card](https://goreportcard.com/badge/github.com/beatlabs/patron)](https://goreportcard.com/report/github.com/beatlabs/patron) [![GoDoc](https://godoc.org/github.com/beatlabs/patron?status.svg)](https://godoc.org/github.com/beatlabs/patron) ![GitHub release](https://img.shields.io/github/release/beatlabs/patron.svg)[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fbeatlabs%2Fpatron.svg?type=shield&issueType=license)](https://app.fossa.com/projects/git%2Bgithub.com%2Fbeatlabs%2Fpatron?ref=badge_shield&issueType=license)

Patron is a framework for creating microservices, originally created by Sotiris Mantzaris (<https://github.com/mantzas>). This fork is maintained by Beat Engineering (<https://thebeat.co>)

`Patron` is french for `template` or `pattern`, but it means also `boss` which we found out later (no pun intended).

The entry point of the framework is the `Service`. The `Service` uses `Components` to handle the processing of sync and async requests. The `Service` starts by default an `HTTP Component` which hosts the `/debug`, `/alive`, `/ready` and `/metrics` endpoints. Any other endpoints will be added to the default `HTTP Component` as `Routes`. Alongside `Routes` one can specify middleware functions to be applied ordered to all routes as `MiddlewareFunc`. The service sets up by default logging with `slog`, tracing and metrics with [OpenTelemetry](https://opentelemetry.io).

`Patron` provides abstractions for the following functionality of the framework:

- service, which orchestrates everything
- components and processors, which provide an abstraction of adding processing functionality to the service
  - asynchronous message processing (RabbitMQ, Kafka, AWS SQS)
  - synchronous processing (HTTP)
  - gRPC support
- metrics and tracing
- logging

`Patron` provides the sane defaults for making the usage as simple as possible.
`Patron` needs Go 1.24 as a minimum.

## Code coverage

![Code coverage](https://codecov.io/gh/beatlabs/patron/graphs/icicle.svg?token=sxY15rXW1X)

## Table of Contents

- [Code of Conduct](docs/CodeOfConduct.md)
- [Contribution Guidelines](docs/ContributionGuidelines.md)
- [Acknowledgments](docs/ACKNOWLEDGMENTS.md)

## API docs

- Components
  - [AMQP component](docs/api/components/amqp.md)
  - [Kafka component](docs/api/components/kafka.md)
  - [gRPC component](docs/api/components/grpc.md)
  - [SQS component](docs/api/components/sqs.md)
- Clients
  - [AMQP client](docs/api/clients/amqp.md)
  - [Kafka client](docs/api/clients/kafka.md)
  - [gRPC client](docs/api/clients/grpc.md)
  - [SQS client](docs/api/clients/sqs.md)
