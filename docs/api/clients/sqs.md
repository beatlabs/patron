# SQS client (AWS SQS)

Use your AWS SDK v2 SQS client directly; Patron provides a consumer component but no custom SQS client wrapper. The example app shows how to create the AWS client and send messages.

- Example producer: `examples/client/main.go` (sendSQSMessage)
- Example consumer: `examples/service/sqs.go`

## Quick start (producer)

```go
cfg, _ := examples.CreateSQSConfig(ctx)
client := sqs.NewFromConfig(cfg, sqs.WithEndpointResolverV2(&examples.SQSCustomResolver{}))
url, _ := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: aws.String(queue)})
_, _ = client.SendMessage(ctx, &sqs.SendMessageInput{QueueUrl: url.QueueUrl, MessageBody: aws.String("hello")})
```

Tracing is enabled through the general OpenTelemetry setup in this repo (see `observability/`), plus aws-sdk-v2 instrumentation vendored with the project.
