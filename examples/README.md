# example

The following example will show off the usage of patron involving all components implemented.
The processing will be kicked of by sending a request to the HTTP component. The flow then will be the following:

- HTTP -> RabbitMQ publish
- RabbitMQ consumer -> kafka publish
- Kafka consumer -> log to stdout
- Publish message to AWS SQS and SNS -> Consume them with AWS SQS. On this step, the same message
is sent to both the SQS queue and the SNS topic, so as to show you how to create both SNS and SQS producers.

Since tracing instrumentation is in place we can observer the flow in Jaeger.

## Prerequisites

- Docker
- Docker compose

## Setting up environment

To run the full example we need to start [jaeger](https://www.jaegertracing.io/) and [prometheus](https://prometheus.io/). We can startup both of them using docker-compose with the following command.

```shell
docker-compose up -d
```

To tear down the above just:

```shell
docker-compose down
```

## Running the examples

When the services started with Docker Compose are ready, you will need to start each of the five
examples in order:

```shell
go run examples/first/main.go
go run examples/second/main.go
go run examples/third/main.go
go run examples/fourth/main.go
go run examples/fifth/main.go
```

and then send a sample request:

```shell
./start_processing.sh
```

After that head over to [jaeger](http://localhost:16686/search) and [prometheus](http://localhost:9090/graph).