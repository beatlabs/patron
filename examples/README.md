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
make deps-start
```

Next we run the service:

```bash
make example-service
```

and afterwards the client:

```bash
make example-client
```
