# AWS SQS

## Description

The SQS component is an easy way to introduce AWS SQS in our service in order to process messages out of a queue.  
The component needs only to be provided with a process function `type ProcessorFunc func(context.Context, Batch)`  
The component utilizes the official [AWS SDK for Go](http://github.com/aws/aws-sdk-go-v2/).  
The component is able to process messages from the queue in a batch mode (as the SDK also provides).  
Messages are either acknowledged as a batch, or we can acknowledge them individually.
To get a head start you can go ahead and take a look at the [sqs example](/examples/sqs/main.go) for a hands-on demonstration of the SQS package in the context of collaborating Patron components.

### Message

The message interface contains methods for:

- getting the context and from it an associated logger
- getting the raw SQS message
- getting the span of the distributed trace
- acknowledging a message
- not acknowledging a message

### Batch

The batch interface contains methods for:

- getting all messages of the batch
- acknowledging all messages in the batch with a single SDK call
- not acknowledging the batch

## Concurrency

Handling messages sequentially or concurrently is left to the process function supplied by the developer.

## Observability

The package collects Prometheus metrics regarding the queue usage. These metrics are about the message age, the queue size, the total number of messages, as well as how many of them were delayed or not visible (in flight).
The package has also included distributed trace support OOTB.
