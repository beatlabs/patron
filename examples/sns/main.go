package main

import (
	"context"
	"errors"
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
	patronsns "github.com/beatlabs/patron/client/sns"
	patronsqs "github.com/beatlabs/patron/client/sqs"
	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/component/async/amqp"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	oamqp "github.com/streadway/amqp"
)

const (
	amqpURL          = "amqp://guest:guest@localhost:5672/"
	amqpQueue        = "patron"
	amqpExchangeName = "patron"
	amqpExchangeType = oamqp.ExchangeFanout

	// Shared AWS config
	awsRegion = "eu-west-1"
	awsID     = "test"
	awsSecret = "test"
	awsToken  = "token"

	// SQS config
	awsSQSEndpoint = "http://localhost:4566"
	awsSQSQueue    = "patron"

	// SNS config
	awsSNSEndpoint = "http://localhost:4566"
	awsSNSTopic    = "patron-topic"
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
	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", "50003")
	if err != nil {
		fmt.Printf("failed to set default patron port env vars: %v", err)
		os.Exit(1)
	}
}

func main() {
	name := "sns"
	version := "1.0.0"

	service, err := patron.New(name, version)
	if err != nil {
		fmt.Printf("failed to set up service: %v", err)
		os.Exit(1)
	}

	// Programmatically create an empty SQS queue for the sake of the example
	sqsAPI := sqs.New(getAWSSession(awsSQSEndpoint))
	sqsQueueURL, err := createSQSQueue(sqsAPI)
	if err != nil {
		log.Fatalf("failed to create sqs queue: %v", err)
	}

	// Programmatically create an SNS topic for the sake of the example
	snsAPI := sns.New(getAWSSession(awsSNSEndpoint))
	snsTopicArn, err := createSNSTopic(snsAPI)
	if err != nil {
		log.Fatalf("failed to create sns topic: %v", err)
	}

	// Route the SNS topic to the SQS queue, so that any message received on the SNS topic
	// will be automatically sent to the SQS queue.
	err = routeSNSTOpicToSQSQueue(snsAPI, sqsQueueURL, snsTopicArn)
	if err != nil {
		log.Fatalf("failed to route sns to sqs: %v", err)
	}

	// Create an SNS publisher
	snsPub, err := patronsns.NewPublisher(snsAPI)
	if err != nil {
		log.Fatalf("failed to create sns publisher: %v", err)
	}

	// Create an SQS publisher
	sqsPub, err := patronsqs.NewPublisher(sqsAPI)
	if err != nil {
		log.Fatalf("failed to create sqs publisher: %v", err)
	}

	// Initialise the AMQP component
	amqpCmp, err := newAmqpComponent(
		amqpURL,
		amqpQueue,
		amqpExchangeName,
		amqpExchangeType,
		[]string{"bind.one.*", "bind.two.*"},
		snsTopicArn,
		snsPub,
		sqsPub,
		sqsQueueURL,
	)
	if err != nil {
		log.Fatalf("failed to create processor %v", err)
	}

	ctx := context.Background()
	err = service.WithComponents(amqpCmp.cmp).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}

func getAWSSession(endpoint string) *session.Session {
	return session.Must(
		session.NewSession(
			&aws.Config{
				Region:      aws.String(awsRegion),
				Credentials: credentials.NewStaticCredentials(awsID, awsSecret, awsToken),
			},
			&aws.Config{Endpoint: aws.String(endpoint)},
		),
	)
}

func createSQSQueue(api sqsiface.SQSAPI) (string, error) {
	out, err := api.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(awsSQSQueue),
	})
	if out.QueueUrl == nil {
		return "", errors.New("could not create the queue")
	}
	return *out.QueueUrl, err
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

func routeSNSTOpicToSQSQueue(snsAPI snsiface.SNSAPI, sqsQueueArn, topicArn string) error {
	_, err := snsAPI.Subscribe(&sns.SubscribeInput{
		Protocol: aws.String("sqs"),
		TopicArn: aws.String(topicArn),
		Endpoint: aws.String(sqsQueueArn),
		Attributes: map[string]*string{
			// Set the RawMessageDelivery to "true" in order to be able to pass the MessageAttributes from SNS
			// to SQS, and therefore to propagate the trace.
			// See https://docs.aws.amazon.com/sns/latest/dg/sns-message-attributes.html for more information.
			"RawMessageDelivery": aws.String("true"),
		},
	})

	return err
}

type amqpComponent struct {
	cmp         patron.Component
	snsTopicArn string
	snsPub      patronsns.Publisher
	sqsPub      patronsqs.Publisher
	sqsQueueURL string
}

func newAmqpComponent(url, queue, exchangeName, exchangeType string, bindings []string, snsTopicArn string, snsPub patronsns.Publisher,
	sqsPub patronsqs.Publisher, sqsQueueURL string) (*amqpComponent, error) {
	amqpCmp := amqpComponent{
		snsTopicArn: snsTopicArn,
		snsPub:      snsPub,
		sqsPub:      sqsPub,
		sqsQueueURL: sqsQueueURL,
	}

	exchange, err := amqp.NewExchange(exchangeName, exchangeType)
	if err != nil {
		return nil, err
	}

	cf, err := amqp.New(url, queue, *exchange, amqp.Bindings(bindings...))
	if err != nil {
		return nil, err
	}

	cmp, err := async.New("amqp-cmp", cf, amqpCmp.Process).
		WithRetries(10).
		WithRetryWait(10 * time.Second).
		Create()
	if err != nil {
		return nil, err
	}
	amqpCmp.cmp = cmp

	return &amqpCmp, nil
}

func (ac *amqpComponent) Process(msg async.Message) error {
	var u examples.User

	err := msg.Decode(&u)
	if err != nil {
		return err
	}

	payload, err := json.Encode(u)
	if err != nil {
		return err
	}

	// Create a new SNS message and publish it
	snsMsg, err := patronsns.NewMessageBuilder().
		Message(string(payload)).
		TopicArn(ac.snsTopicArn).
		Build()
	if err != nil {
		return fmt.Errorf("failed to create message: %v", err)
	}
	_, err = ac.snsPub.Publish(msg.Context(), *snsMsg)
	if err != nil {
		return fmt.Errorf("failed to publish message to SNS: %v", err)
	}

	// Create a new SQS message and publish it
	sqsMsg, err := patronsqs.NewMessageBuilder().
		Body(string(payload)).
		QueueURL(ac.sqsQueueURL).
		Build()
	_, err = ac.sqsPub.Publish(msg.Context(), *sqsMsg)
	if err != nil {
		return fmt.Errorf("failed to publish message to SQS: %v", err)
	}

	log.FromContext(msg.Context()).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return nil
}
