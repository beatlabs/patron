package main

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/beatlabs/patron"
	patronsqs "github.com/beatlabs/patron/component/sqs"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
)

func createSQSConsumer() (patron.Component, error) {
	process := func(_ context.Context, btc patronsqs.Batch) {
		for _, msg := range btc.Messages() {
			err := msg.ACK()
			if err != nil {
				log.FromContext(msg.Context()).Info("AWS SQS message received but ack failed", "msgID", msg.ID(), "error", err)
				continue
			}
			log.FromContext(msg.Context()).Info("AWS SQS message received and acked", "msgID", msg.ID())
		}
	}

	api, err := examples.CreateSQSAPI()
	if err != nil {
		return nil, err
	}

	out, err := api.CreateQueue(context.Background(), &sqs.CreateQueueInput{
		QueueName: aws.String(examples.AWSSQSQueue),
	})
	if err != nil {
		return nil, err
	}
	if out.QueueUrl == nil {
		return nil, errors.New("could not create the queue")
	}

	return patronsqs.New("sqs-cmp", examples.AWSSQSQueue, api, process, patronsqs.WithPollWaitSeconds(5))
}
