package main

import (
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
	"github.com/beatlabs/patron/async/amqp"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	patronsns "github.com/beatlabs/patron/trace/sns"
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
	awsSQSEndpoint = "http://localhost:4576"
	awsSQSQueue    = "patron"

	// SNS config
	awsSNSEndpoint = "http://localhost:4575"
	awsSNSTopic    = "patron-topic"
)

var (
	awsSQSSession *session.Session
	awsSNSSession *session.Session

	amqpBindings = []string{"bind.one.*", "bind.two.*"}
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

	baseConfig := &aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsID, awsSecret, awsToken),
	}

	awsSNSSession = session.Must(
		session.NewSession(
			baseConfig,
			&aws.Config{Endpoint: aws.String(awsSNSEndpoint)},
		),
	)

	awsSQSSession = session.Must(
		session.NewSession(
			baseConfig,
			&aws.Config{Endpoint: aws.String(awsSQSEndpoint)},
		),
	)
}

func main() {
	name := "fourth"
	version := "1.0.0"

	err := patron.Setup(name, version)
	if err != nil {
		fmt.Printf("failed to set up logging: %v", err)
		os.Exit(1)
	}

	// Programmatically create an empty SQS queue for the sake of the example
	sqsAPI := sqs.New(awsSQSSession)
	sqsQueueArn, err := createSQSQueue(sqsAPI)
	if err != nil {
		log.Fatalf("failed to create sqs queue: %v", err)
	}

	// Programmatically create an SNS topic for the sake of the example
	snsAPI := sns.New(awsSNSSession)
	snsTopicArn, err := createSNSTopic(snsAPI)
	if err != nil {
		log.Fatalf("failed to create sns topic: %v", err)
	}

	// Route the SNS topic to the SQS queue, so that any message received on the SNS topic
	// will be automatically sent to the SQS queue.
	err = routeSNSTOpicToSQSQueue(snsAPI, sqsQueueArn, snsTopicArn)
	if err != nil {
		log.Fatalf("failed to route sns to sqs: %v", err)
	}

	// Create an SNS publisher
	snsPub, err := patronsns.NewPublisher(snsAPI)
	if err != nil {
		log.Fatalf("failed to create sns publisher: %v", err)
	}

	// Initialise the AMQP component
	amqpCmp, err := newAmqpComponent(amqpURL, amqpQueue, amqpExchangeName, amqpExchangeType, amqpBindings, snsTopicArn, snsPub)
	if err != nil {
		log.Fatalf("failed to create processor %v", err)
	}

	srv, err := patron.New(
		name,
		version,
		patron.Components(amqpCmp.cmp),
	)
	if err != nil {
		log.Fatalf("failed to create service %v", err)
	}

	err = srv.Run()
	if err != nil {
		log.Fatalf("failed to run service %v", err)
	}
}

func createSQSQueue(api sqsiface.SQSAPI) (string, error) {
	out, err := api.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(awsSQSQueue),
	})
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
}

func newAmqpComponent(url, queue, exchangeName, exchangeType string, bindings []string, snsTopicArn string, snsPub patronsns.Publisher) (*amqpComponent, error) {
	amqpCmp := amqpComponent{
		snsTopicArn: snsTopicArn,
		snsPub:      snsPub,
	}

	exchange, err := amqp.NewExchange(exchangeName, exchangeType)

	if err != nil {
		return nil, err
	}

	cf, err := amqp.New(url, queue, *exchange, amqp.Bindings(bindings...))
	if err != nil {
		return nil, err
	}

	cmp, err := async.New("amqp-cmp", amqpCmp.Process, cf, async.ConsumerRetry(10, 10*time.Second))
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

	snsMsg, err := patronsns.NewMessageBuilder().
		Message(string(payload)).
		TopicArn(ac.snsTopicArn).
		Build()
	if err != nil {
		return fmt.Errorf("failed to create message: %v", err)
	}

	_, err = ac.snsPub.Publish(msg.Context(), *snsMsg)
	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	log.FromContext(msg.Context()).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return nil
}
