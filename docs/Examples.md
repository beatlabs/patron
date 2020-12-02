# Examples

The [examples/](/examples) folder contains a number of small services which work together to provide an overview of Patron's clients and components, as well as the built-in tracing and logging. After you start them all, you can see how a request travels through all services by triggering the `start_processing.sh` script.

## Prerequisites

Before all services start we should first start all dependencies with `docker-compose`.

```shell
docker-compose up -d
```

To tear down the above just:

```shell
docker-compose down
```

## [HTTP Cache Service](../examples/http-cache/main.go) 

The service shows how to use:
 
- HTTP caching in a specific route using Redis
- Textual logger 
  
The service can be started as follows:

```shell
go run examples/http-cache/main.go
```

## [HTTP Service](../examples/http/main.go)

The service shows how to use:
 
- HTTP with CORS
- HTTP client with API key
- Textual logger with predefined fields
  
The service can be started as follows:

```shell
go run examples/http/main.go
```

## [HTTP API Key Service](../examples/http-sec/main.go)

- HTTP service with a secured route (API KEY)
- Async Kafka publisher
- Default structured logger with predefined fields

The service can be started as follows:

```shell
go run examples/http-sec/main.go
```

## [Kafka Service](../examples/kafka/main.go)

The service shows how to use:

- Kafka with a group consumer
- AMQP publisher
- Textual logger

The service can be started as follows:

```shell
go run examples/amqp/main.go
```

## [AMQP Service](../examples/kafka/main.go)

The service shows how to use:

- AMQP consumer
- AWS SNS Publisher
- AWS SQS Publisher
- Default structured logger

The service can be started as follows:

```shell
go run examples/sns/main.go
```

## [AWS SQS Service](../examples/sqs/main.go)

The service shows how to use:

- AWS SQS Consumer
- gRPC client
- Default structured logger

The service can be started as follows:

```shell
go run examples/sqs/main.go
```

## [AWS SQS Concurrent Service](../examples/sqs-simple/main.go)

The service shows how to use:

- AWS SQS Concurrent Consumer
- Default structured logger

The service can be started as follows:

```shell
go run examples/sqs-simple/main.go
```

## [gRPC Service](../examples/grpc/main.go)

The service shows how to use:

- gRPC Server
- Textual logger

The service can be started as follows:

```shell
go run examples/grpc/main.go
```

## All of the above working together

After all services have been started successfully we can send a request and see how it travels through all of them by running. 

```shell
../examples/start_processing.sh
```

After that head over to [jaeger](http://localhost:16686/search) and [prometheus](http://localhost:9090/graph).


## [Compression Middleware](../examples/compression-middleware)
The compression-middleware example showcases the compression middleware with a /foo route that returns some random data.
```shell
$ go run examples/compression-middleware/main.go 
$ curl -s localhost:50000/foo | wc -c
1398106
$ curl -s localhost:50000/foo -H "Accept-Encoding: nonexisting" | wc -c
1398106
$ curl -s localhost:50000/foo -H "Accept-Encoding: gzip" | wc -c
1053068
$ curl -s localhost:50000/foo -H "Accept-Encoding: deflate" | wc -c
1053045
```

It also contains a /hello route used by the next example

## [Client Decompression](../examples/client-decompression)
After launching the `compression-middleware` example, you can run the following to validate that Patron's HTTP client
handles compressed requests transparently. 

It creates three requests (with and without an 'Accept-Encoding' header), where you can
see that the response from the previous example is decompressed automatically.

```shell
go run examples/client-decompression/main.go
```