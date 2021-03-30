# AMQP

## Description

The AMQP component is an easy way to introduce AMQP in our service in order to process messages out of a queue.  
- The component needs only a process function `type ProcessorFunc func(Batch)` to be provided.  
- The component utilizes the [Streadway's AMQP](http://github.com/streadway/amqp) package.  
- The component is able to process messages from the queue in a batch mode.  
- Messages are either acknowledged as a batch, or we can acknowledge them individually.
- The component is also able to handle AMQP failures with retries.
To get a head start you can go ahead and take a look at the [AMQP example](/examples/amqp/main.go) for a hands-on demonstration of the AMQP package in the context of collaborating Patron components.

### Message

The message interface contains methods for:

- getting the context and from it an associated logger
- getting the raw AMQP message
- getting the span of the distributed trace
- acknowledging a message
- not acknowledging a message

### Batch

The batch interface contains methods for:

- getting all messages of the batch
- acknowledging all messages in the batch
- not acknowledging the batch

## Concurrency

Handling messages sequentially or concurrently is left to the process function supplied by the developer.

## Observability

The package collects Prometheus metrics regarding the queue usage. These metrics are about the queue size, 
the total number of messages received and which of them we acknowledge and not.
The package has also included distributed trace support OOTB.
 