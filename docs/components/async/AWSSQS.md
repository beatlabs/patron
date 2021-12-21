# AWS SQS

## Deprecated

The SQS consumer package along with the async component is superseded by the standalone `github.com/beatlabs/patron/component/sqs` package.

**This package is frozen and no new functionality will be added.**

## Description

The SQS component allows users to construct SQS consumers and handle messages under the `async.Message` abstraction. It supports JSON and Protobuf-encoded messages.

The package collects Prometheus metrics regarding the queue usage. These metrics are about the message age, the queue size, the total number of messages, as well as how many of them were delayed or not visible (in flight).

Users can configure

- the maximum number of messages to [fetch at once](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/APIReference/API_ReceiveMessage.html)
- the use of [short- or long-polling](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-short-and-long-polling.html)
- the wait time for the long-polling mechanism
- the message [visibility timeout](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html)
- the buffer size for consuming messages concurrently
- the interval at which stats are collected

The component utilizes the official [AWS SDK for Go](http://github.com/aws/aws-sdk-go/); to get a head start you can go ahead and take a look at the [sqs example](/examples/sqs/main.go) for a hands-on demonstration of the SQS package in the context of collaborating Patron components.

As with all Patron components, tracing capabilities are included out of the box.
