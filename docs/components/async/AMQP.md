# AMQP

The AMQP component allows users to construct consumers for AMQP-based queues. It also provides helper functions for working with consumed messages and the `async.Message` abstraction. The consumer supports JSON and Protobuf-encoded messages.

The supported [exchange types](https://www.rabbitmq.com/tutorials/amqp-concepts.html#exchanges) are four; *direct*, *fanout*, *topic* and *header*.

Users can configure the incoming messages buffer size, the connection timeout, whether rejected message should be requeued, as well as provide custom exchange-queue [bindings](https://www.rabbitmq.com/tutorials/amqp-concepts.html#bindings).

The AMQP consumer component is powered by the battle-tested [`streadway/amqp`](https://www.rabbitmq.com/tutorials/amqp-concepts.html#bindings) package. In the [examples](/examples/amqp/main.go) folder you can see the component in action.

As with all Patron components, tracing capabilities are included out of the box.
