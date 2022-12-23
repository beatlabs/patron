package main

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/beatlabs/patron"
	patronsqs "github.com/beatlabs/patron/component/sqs"
	"github.com/beatlabs/patron/log"
)

const (
	awsRegion      = "eu-west-1"
	awsID          = "test"
	awsSecret      = "test"
	awsToken       = "token"
	awsSQSEndpoint = "http://localhost:4566"
	awsSQSQueue    = "patron"
)

func createSQSConsumer() (patron.Component, error) {
	process := func(_ context.Context, btc patronsqs.Batch) {
		for _, msg := range btc.Messages() {
			log.FromContext(msg.Context()).Infof("AWS SQS message received: %s", msg.ID())

			msg.NACK()
		}
	}

	api, err := createSQSAPI(awsSQSEndpoint)
	if err != nil {
		return nil, err
	}

	out, err := api.CreateQueue(context.Background(), &sqs.CreateQueueInput{
		QueueName: aws.String(awsSQSQueue),
	})
	if err != nil {
		return nil, err
	}
	if out.QueueUrl == nil {
		return nil, errors.New("could not create the queue")
	}

	return patronsqs.New("sqs-cmp", awsSQSQueue, api, process, patronsqs.WithPollWaitSeconds(5))
}

func createSQSAPI(endpoint string) (*sqs.Client, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == sqs.ServiceID && region == awsRegion {
			return aws.Endpoint{
				URL:           endpoint,
				SigningRegion: awsRegion,
			}, nil
		}
		// returning EndpointNotFoundError will allow the service to fallback to it's default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(awsID, awsSecret, awsToken))),
	)
	if err != nil {
		return nil, err
	}

	api := sqs.NewFromConfig(cfg)

	return api, nil
}
