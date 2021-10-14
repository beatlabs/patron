# Clients

Patron microservices can interact with other microservices, APIs and applications using a number of clients.

All clients contain integrated tracing powered by `opentracing-go`; any new clients should attempt to do the same.

**Third-party dependencies**  
github.com/opentracing/opentracing-go v1.1.0


## HTTP Client
Patron provides an HTTP client which integrates tracing into all outgoing requests by wrapping the default `net/http` client. 
Users can configure the client's Timeout, RoundTripper and/or set up a circuit breaker. 
In order to propagate the traces, the HTTP request context needs to be set.

## AMQP
The AMQP client allows users to connect to a RabbitMQ instance and publish messages. The published messages have integrated tracing headers by default. Users can configure every aspect of the connection.

**Third-party dependencies**  
github.com/streadway/amqp v0.0.0-20180315184602-8e4aba63da9f

## gRPC
The gRPC client initiates a client connection to a given target while injecting a `UnaryInterceptor` to integrate tracing capabilities. By default, this is a non-blocking connection and users can pass in any number of [`grpc.DialOption`](https://github.com/grpc/grpc-go/blob/master/dialoptions.go) arguments to configure its behavior.

**Third-party dependencies**  
google.golang.org/grpc v1.27.1


## Kafka
The Kafka client allows users to create a synchronous or asynchronous Kafka producer and publish Kafka messages with tracing headers. The builder pattern allows users to configure every aspect of the connection.

**Third-party dependencies**  
github.com/Shopify/sarama v1.30.0


## Redis
The Redis client allows users to connect to a Redis instance and execute commands. The connection can be configured using [`redis.Options`](https://github.com/go-redis/redis/blob/v7/options.go).

**Third-party dependencies**  
github.com/go-redis/redis/v7 v7.0.0-beta.5


## SQL
The SQL client enhances the standard library SQL by integrating tracing capabilities. It has support for prepared statements, queries, as well as low-level handling of transactions.


## SNS - SQS
The SNS and SQS clients provide wrappers useful for publishing messages to AWS SNS and SQS, with integrating tracing.

**Third-party dependencies**  
github.com/aws/aws-sdk-go v1.21.8


## Elasticsearch
The Elasticsearch client allows users to connect to an elasticsearch instance. Its behavior can be configured by providing an [`elasticsearch.Config`](https://github.com/elastic/go-elasticsearch/blob/4b40206692088570801280584e614027e6ce818b/elasticsearch.go#L32) struct

**Third-party dependencies**  
github.com/elastic/go-elasticsearch/v8 v8.0.0-20190731061900-ea052088db25

