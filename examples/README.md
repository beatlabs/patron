# example

The following example will show off the usage of patron involving all components implemented.
The processing will be kicked of by sending a request to the HTTP component. The flow then will be the following:

- HTTP -> RabbitMQ publish
- RabbitMQ consumer -> kafka publish
- Kafka consumer -> log to stdout
- Publish message to AWS SNS -> Consume it with AWS SQS

Since tracing instrumentation is in place we can observer the flow in jaeger.

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

Start first service:

```shell
go run examples/first/main.go
```

Start second service:

```shell
go run examples/second/main.go

```

Start third service:

```shell
go run examples/third/main.go
```

Start fourth service:

```shell
go run examples/fourth/main.go
```

Start fifth service:

```shell
go run examples/fifth/main.go
```

and the use curl to send a request:

```shell
curl -d '{"Firstname":"John", "Lastname": "Doe"}' -H "Content-Type: application/json" -X POST http://localhost:50000
```

After that head over to [jaeger](http://localhost:16686/search) and [prometheus](http://localhost:9090/graph).