---
layout: default
title: SQS component
parent: Components
nav_order: 4
---

# SQS component

The SQS component allows you to create a consumer for an AWS SQS queue.

## Creation

A new SQS component can be created using the constructor `sqs.New()`. The constructor requires a component name, the SQS queue name, an SQS API implementation, and a processor function. The processor function is of type `sqs.ProcessorFunc` and is responsible for processing a batch of one or more SQS messages.

```go
// Create a new SQS API implementation.
// The AWS session is omitted for brevity.
api := sqs.NewFromConfig(awsCfg)

// Create a new SQS component.
cmp, err := sqs.New("my-sqs-consumer", "my-queue", api,
    func(ctx context.Context, b sqs.Batch) {
        // Process the batch of messages.
        for _, msg := range b.Messages() {
            // Get the message body.
            body := msg.Body()
            // Process the message.
            // ...
            // Acknowledge the message.
            msg.ACK()
        }
    },
)
```

## Options

The SQS component can be configured using one or more of the following options:

- `sqs.WithMaxMessages()`: Sets the maximum number of messages to fetch in a single request. The default value is 3, and the maximum is 10.
- `sqs.WithPollWaitSeconds()`: Sets the long-polling interval in seconds. The default is 0 (short polling), and the maximum is 20.
- `sqs.WithVisibilityTimeout()`: Sets the visibility timeout for messages. The default is 0, and the maximum is 12 hours.
- `sqs.WithQueueStatsInterval()`: Sets the interval for reporting queue statistics. The default is 10 seconds.
- `sqs.WithRetries()`: Sets the number of retries for fetching messages. The default is 10.
- `sqs.WithRetryWait()`: Sets the wait time between retries. The default is 1 second.
- `sqs.WithQueueOwner()`: Sets the AWS account ID of the queue owner.

## Metrics

The SQS component exposes the following metrics:

- `sqs.message.age`: The age of the oldest message in the queue in seconds.
- `sqs.message.counter`: The number of messages processed, with a `state` label (`ACK`, `NACK`, `FETCHED`).
- `sqs.queue.size`: The number of messages in the queue, with a `state` label (`available`, `delayed`, `invisible`).
