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

## Migrating from `aws-sdk-go` v1 to v2

For leveraging the AWS patron components updated to `aws-sdk-go` v2, the client initialization should be modified. In v2 the [session](https://docs.aws.amazon.com/sdk-for-go/api/aws/session) package was replaced with a simple configuration system provided by the [config](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/config) package.

Options such as the AWS region and endpoint to be used can be mentioned during configuration loading.

Endpoint resolvers are used to specify custom endpoints for a particular service and region.

The AWS client configured can be plugged in to the respective patron component on initialization, in the same way its predecessor did in earlier patron versions.

An example of configuring a client and plugging it on a patron component can be found [here](../../examples/sqs/main.go).

> A more detailed documentation on migrating can be found [here](https://aws.github.io/aws-sdk-go-v2/docs/migrating).
