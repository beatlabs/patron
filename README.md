# patron ![CI](https://github.com/beatlabs/patron/workflows/CI/badge.svg) [![Coverage Status](https://coveralls.io/repos/github/beatlabs/patron/badge.svg?branch=master)](https://coveralls.io/github/beatlabs/patron?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/beatlabs/patron)](https://goreportcard.com/report/github.com/beatlabs/patron) [![GoDoc](https://godoc.org/github.com/beatlabs/patron?status.svg)](https://godoc.org/github.com/beatlabs/patron) ![GitHub release](https://img.shields.io/github/release/beatlabs/patron.svg)

Patron is a framework for creating microservices, originally created by Sotiris Mantzaris (<https://github.com/mantzas>). This fork is maintained by Beat Engineering (<https://thebeat.co>)

`Patron` is french for `template` or `pattern`, but it means also `boss` which we found out later (no pun intended).

The entry point of the framework is the `Service`. The `Service` uses `Components` to handle the processing of sync and async requests. The `Service` starts by default an `HTTP Component` which hosts the `/debug`, `/alive`, `/ready` and `/metrics` endpoints. Any other endpoints will be added to the default `HTTP Component` as `Routes`. Alongside `Routes` one can specify middleware functions to be applied ordered to all routes as `MiddlewareFunc`. The service sets up by default logging with [zerolog](https://github.com/rs/zerolog), tracing and metrics with [Jaeger](https://www.jaegertracing.io/) and [prometheus](https://prometheus.io/).

`Patron` provides abstractions for the following functionality of the framework:

- service, which orchestrates everything
- components and processors, which provide an abstraction of adding processing functionality to the service
  - asynchronous message processing (RabbitMQ, Kafka, AWS SQS)
  - synchronous processing (HTTP)
  - gRPC support
- metrics and tracing
- logging

`Patron` provides the same defaults for making the usage as simple as possible.
`Patron` needs Go 1.17 as a minimum.

## Table of Contents

- [Architecture](docs/Architecture.md)
- [CLI](docs/other/CLI.md)
- Components
  - [Async](docs/components/async/Async.md)
    - [Kafka](docs/components/async/Kafka.md)
    - [AMQP](docs/components/async/AMQP.md)
    - [AWS SQS (deprecated)](docs/components/async/AWSSQS.md)
  - [HTTP (deprecated)](docs/components/HTTP.md)
  - [HTTP v2](docs/components/HTTPv2.md)
  - [gRPC](docs/components/gRPC.md)
  - [AWS SQS](docs/components/SQS.md)
  - [AMQP](docs/components/AMQP.md)
- [Clients](docs/clients/Clients.md)
- Packages
  - [Observability](docs/observability/Observability.md)
  - [Distributed Tracing](docs/observability/DistributedTracing.md)  
  - [Errors](docs/other/Errors.md)
- [Examples](docs/Examples.md)
- [Code of Conduct](docs/CodeOfConduct.md)
- [Contribution Guidelines](docs/ContributionGuidelines.md)
- [Acknowledgments](docs/ACKNOWLEDGMENTS.md)
