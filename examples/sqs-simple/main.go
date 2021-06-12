package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/async"
	patronsqs "github.com/beatlabs/patron/component/async/sqs"
	"github.com/beatlabs/patron/log"
)

type sqsConfig struct {
	endpoint string
	name     string
	region   string
}

// Make sure localstack is running locally, or point to actual queue on AWS
var sampleConfig = sqsConfig{
	endpoint: "http://localhost:4566",
	name:     "sandbox-payin",
	region:   "eu-west-1",
}

func init() {
	err := os.Setenv("PATRON_LOG_LEVEL", "debug")
	if err != nil {
		fmt.Printf("failed to set log level env var: %v", err)
		os.Exit(1)
	}
}

func main() {
	name := "sqs"
	version := "1.0.0"

	service, err := patron.New(name, version)
	if err != nil {
		fmt.Printf("failed to set up service: %v", err)
		os.Exit(1)
	}
	ctx := context.Background()

	sqsComponent, err := sampleSqs()
	if err != nil {
		log.Fatalf("failed to create sqs component: %v", err)
	}

	err = service.WithComponents(sqsComponent).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service: %v", err)
	}
}

func sampleSqs() (*async.Component, error) {
	sess, err := session.NewSession(&aws.Config{
		Endpoint: &sampleConfig.endpoint,
		Region:   &sampleConfig.region,
	})
	if err != nil {
		return nil, err
	}
	sqsClient := sqs.New(sess)

	factory, err := patronsqs.NewFactory(
		sqsClient,
		sampleConfig.name,
		// Optionally override the queue's default polling setting.
		// Long polling is highly recommended to avoid large costs on AWS.
		// See https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-short-and-long-polling.html
		// It's probably best to not specify any value: the default value on the queue will be used.
		patronsqs.PollWaitSeconds(20),
		// Optionally override the queue's default visibility timeout.
		// See https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html
		// Again, a sensible default should be configured on the queue, but there might be specific use case where you want to override.
		patronsqs.VisibilityTimeout(30),
		// Optionally change the number of messages fetched by each worker.
		// The default is 3.
		patronsqs.MaxMessages(5),
	)
	if err != nil {
		return nil, err
	}

	// Note: the retry count is not increased on an error processing a message, but rather consuming from the queue.
	// If the max number if retries is reached, the service will terminate.
	// The max number of retires of a message is determined by the SQS queue, not the consumer.
	return async.New("sqs", factory, messageHandler).
		// Note that NackExitStrategy does not work with concurrency, so we need to pick either Nack or Ack Strategy
		// Ack strategy is not recommended for SQS: we want failed messages to end up in the dead letter queue
		WithFailureStrategy(async.NackStrategy).
		WithRetries(3).
		WithRetryWait(30 * time.Second).
		WithConcurrency(10).
		Create()
}

func messageHandler(message async.Message) error {
	log.Infof("Received message, payload: %s", string(message.Payload()))
	time.Sleep(3 * time.Second) // useful to see concurrency in action
	return nil
}
