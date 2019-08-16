package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/async"
	patronsqs "github.com/beatlabs/patron/async/sqs"
	"github.com/beatlabs/patron/log"
	patronsns "github.com/beatlabs/patron/trace/sns"
)

const (
	// Shared AWS config
	awsRegion = "eu-west-1"
	awsID     = "test"
	awsSecret = "test"
	awsToken  = "token"

	// SQS config
	awsSQSEndpoint = "http://localhost:4576"
	awsSQSQueue    = "patron"

	// SNS config
	awsSNSEndpoint = "http://localhost:4575"
	awsSNSTopic    = "patron-topic"
)

var (
	awsSQSSession *session.Session
	awsSNSSession *session.Session
)

func init() {
	err := os.Setenv("PATRON_LOG_LEVEL", "debug")
	if err != nil {
		fmt.Printf("failed to set log level env var: %v", err)
		os.Exit(1)
	}
	err = os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "1.0")
	if err != nil {
		fmt.Printf("failed to set sampler env vars: %v", err)
		os.Exit(1)
	}
	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", "50004")
	if err != nil {
		fmt.Printf("failed to set default patron port env vars: %v", err)
		os.Exit(1)
	}

	baseConfig := &aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsID, awsSecret, awsToken),
	}

	awsSQSSession = session.Must(
		session.NewSession(
			baseConfig,
			&aws.Config{Endpoint: aws.String(awsSQSEndpoint)},
		),
	)
	awsSNSSession = session.Must(
		session.NewSession(
			baseConfig,
			&aws.Config{Endpoint: aws.String(awsSNSEndpoint)},
		),
	)
}

func main() {
	name := "fifth"
	version := "1.0.0"

	err := patron.Setup(name, version)
	if err != nil {
		fmt.Printf("failed to set up logging: %v", err)
		os.Exit(1)
	}

	// Initialise SQS
	sqsAPI := sqs.New(awsSQSSession)
	sqsQueueArn, err := createSQSQueue(sqsAPI)
	if err != nil {
		log.Fatalf("failed to create sqs queue: %v", err)
	}

	sqsCmp, err := createSQSComponent(sqsAPI)
	if err != nil {
		log.Fatalf("failed to create sqs component: %v", err)
	}

	// Initialise SNS
	snsAPI := sns.New(awsSNSSession)
	topicArn, err := createSNSTopic(snsAPI)
	if err != nil {
		log.Fatalf("failed to create sns topic: %v", err)
	}
	snsPub, err := patronsns.NewPublisher(snsAPI)
	if err != nil {
		log.Fatalf("failed to create sns publisher: %v", err)
	}

	// Route SNS to SQS (subscriber)
	err = routeSNSToSQS(snsAPI, sqsQueueArn, topicArn)
	if err != nil {
		log.Fatalf("failed to route sns to sqs: %v", err)
	}

	// Run the server
	srv, err := patron.New(name, version, patron.Components(sqsCmp.cmp))
	if err != nil {
		log.Fatalf("failed to create service: %v", err)
	}

	go func() {
		msg, err := patronsns.NewMessageBuilder().
			WithSubject("Hello").
			WithMessage("I am a message that was sent to SQS").
			WithTopicArn(topicArn).
			Build()
		if err != nil {
			log.Fatalf("failed to create message: %v", err)
		}

		_, err = snsPub.Publish(context.Background(), *msg)
		if err != nil {
			log.Fatalf("failed to publish message: %v", err)
		}
	}()

	err = srv.Run()
	if err != nil {
		log.Fatalf("failed to run service: %v", err)
	}
}

type sqsComponent struct {
	cmp patron.Component
}

func createSQSQueue(api sqsiface.SQSAPI) (string, error) {
	out, err := api.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(awsSQSQueue),
	})
	return *out.QueueUrl, err
}

func createSQSComponent(api sqsiface.SQSAPI) (*sqsComponent, error) {
	sqsCmp := sqsComponent{}

	cf, err := patronsqs.NewFactory(api, awsSQSQueue)
	if err != nil {
		return nil, err
	}

	cmp, err := async.New("sqs-cmp", sqsCmp.Process, cf, async.ConsumerRetry(10, 10*time.Second))
	if err != nil {
		return nil, err
	}
	sqsCmp.cmp = cmp

	return &sqsCmp, nil
}

func (ac *sqsComponent) Process(msg async.Message) error {
	var got sns.PublishInput

	err := msg.Decode(&got)
	if err != nil {
		return err
	}

	log.FromContext(msg.Context()).Infof("request processed: %s - %s", *got.Subject, *got.Message)
	return nil
}

func createSNSTopic(snsAPI snsiface.SNSAPI) (string, error) {
	out, err := snsAPI.CreateTopic(&sns.CreateTopicInput{
		Name: aws.String(awsSNSTopic),
	})
	if err != nil {
		return "", err
	}

	return *out.TopicArn, nil
}

func routeSNSToSQS(snsAPI snsiface.SNSAPI, sqsQueueArn, topicArn string) error {
	_, err := snsAPI.Subscribe(&sns.SubscribeInput{
		Protocol: aws.String("sqs"),
		TopicArn: aws.String(topicArn),
		Endpoint: aws.String(sqsQueueArn),
	})

	return err
}
