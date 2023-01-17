# Running the example

The example consists of a service and its client implementation.
The service implementation uses the following components:

- HTTP
- gRPC
- Kafka
- AWS SQS
- AMQP

The client implements all Patron clients for the components used by the service. There is also a flag that allows targeting a specific service component.

## How to run

First we need to start the dependencies of the example by running:

```bash
docker-compose -f examples/docker-compose.yml up -d
```

Next we run the service:

```bash
go run examples/service/*
```

and afterwards the client:

```bash
go run examples/client/main.go
```

The client is able to target specific server components with a flag. Run the following argument to see what is available:

```bash
go run examples/client/main.go --help
```
