/*
Package patron

Patron is a Go microservice framework. The entrypoint is the Service, which
orchestrates one or more Components implementing Run(context.Context) error.

Defaults and lifecycle

  - A default HTTP component is started to host management endpoints:
    /debug, /alive, /ready, /metrics. Additional routes can be added.
  - Observability (structured logging via slog, OpenTelemetry traces and metrics)
    is set up on Service construction and shut down when the Service stops.

Programming model

  - Components: long-running units with a Run(ctx) error method. The Service
    runs each component in a goroutine and aggregates errors.
  - Synchronous components: HTTP, gRPC.
  - Asynchronous components: AMQP (RabbitMQ), Kafka, AWS SQS.

Key packages

  - component/http, component/grpc, component/kafka, component/amqp, component/sqs
  - client/* packages for instrumented clients (HTTP, gRPC, Kafka, AMQP, SQS,
    Elasticsearch, MongoDB, MQTT, Redis, SNS, SQL)

Usage sketch

	svc, err := patron.New("example", "v1.0.0")
	if err != nil { // handle error }
	// Build components (e.g., HTTP router, Kafka consumer, etc.) and run:
	// err = svc.Run(ctx, httpComponent, kafkaComponent)

Patron provides sane defaults and follows Go idioms. See the repository for
examples and detailed API docs: https://github.com/beatlabs/patron.
*/
package patron
