# Distributed Tracing

Modern distributed software architectures (such as microservices) enable team autonomy and provide enormous scaling capabilities.

At the same time, they introduce new issues with debugging and monitoring of applications; for example it can be extremely hard to diagnose why a request spanning multiple microservices is sometimes slower than usual, or outright fails.

In many cases, [distributed tracing](https://opentracing.io/docs/overview/what-is-tracing/) can help pinpoint where failures occur and what causes poor performance.

Implementing distributed tracing means adding instrumentation to your application code; in the context of multiple microservices owned by different teams, this can pose a big challenge.

One of Patron's goals is to enable uniformity between microservices and allow end-users to focus on their application code. For this reason, *all* of Patron's components and clients contain built-in tracing, from SQL transactions, to asynchronous producing and consuming of messages, HTTP and gRPC calls and caching mechanisms, distributed tracing is ubiquitous throughout Patron.

To see this in action, one can refer to the `examples/` folder; the examples built a chain of seven services, showcasing the majority of Patron's components and clients. The entrypoint is the `start_processing.sh` script. 

```
$ cd patron/examples
$ docker-compose up -d
$ go run http/main.go &
$ go run kafka/main.go &
$ go run amqp/main.go & 
$ go run grpc/main.go &
$ go run http-cache/main.go &
$ go run http-sec/main.go &
$ go run sqs/main.go &
$ ./start_processing.sh
``` 

After running these commands, you can visit the Jaeger client at `localhost:16686/search` and see how you can make use of distributed tracing to debug and optimize your code in complex, distributed systems.

We make use of the battle-tested OpenTracing specification and client, a CNCF project used in production by many tech giants. If you wish to better understand how Distributed Tracing works, you can refer to the official [OpenTracing docs](https://opentracing.io/docs/overview/), read about [spans](https://opentracing.io/docs/overview/spans/) which make up the primary building block of a distributed trace, see how spans work in a [concurrent system](https://opentracing.io/docs/overview/scopes-and-threading/), as well as how spans are [injected and extracted](https://opentracing.io/docs/overview/inject-extract/) to and from carriers.
