# Async

The component is responsible setting up a consumer using the consumer factory, fetching messages from the underlying system and handling the processing of the messages. In case of success the component acknowledges the message and moves to the next. When a message fails to be processed the component will execute the failure strategy setup. The component has also setup logging, capturing of metrics and distributed tracing.

The component makes use of the `Builder` pattern, and expects a consumer factory and a processor function but also provides additional setup methods for the failure strategy, retries, etc.

## Consumer and Factory

The component uses the consumer in order to get messages from the Message Broker/Stream.  
The concrete implementation follows the interface:

```go
// Consumer interface which every specific consumer has to implement.
type Consumer interface {
    Consume(context.Context) (<-chan Message, <-chan error, error)
    Close() error
}
```

The component accepts a factory in order to be able to recreate the consumer when there is need for it. The implementation follows the interface.

```go
// ConsumerFactory interface for creating consumers.
type ConsumerFactory interface {
    Create() (Consumer, error)
}
```

## Processor function

The actual processing of the function that needs to be provided is following the type:

```go
// ProcessorFunc definition of a async processor.
type ProcessorFunc func(Message) error
```

It accepts a `Message` and returns either a nil for success or an error in order to be handled by the failure strategy.

## Message

The messages of the component should follow the interface:

```go
type Message interface {
    Context() context.Context
    Decode(v interface{}) error
    Ack() error
    Nack() error
    Source() string
    Payload() []byte
}
```

## Failure Strategy

The failure strategy defines how the system will behave during the processing of a message.  
The following strategies are available:

- `NackExitStrategy` does not acknowledge the message and exits the application on error
- `NackStrategy` does not acknowledge the message, leaving it for reprocessing, and continues
- `AckStrategy` acknowledges message and continues
