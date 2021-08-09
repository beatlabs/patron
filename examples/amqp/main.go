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
	patronsns "github.com/beatlabs/patron/client/sns/v2"
	patronsqs "github.com/beatlabs/patron/client/sqs/v2"
	patronamqp "github.com/beatlabs/patron/component/amqp"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	"github.com/streadway/amqp"
)

const (
	amqpURL          = "amqp://guest:guest@localhost:5672/"
	amqpQueue        = "patron"
	amqpExchangeName = "patron"
	amqpExchangeType = amqp.ExchangeFanout

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

	// Setup queue and exchange if not already done.
	err = setupQueueAndExchange()
	if err != nil {
		fmt.Printf("failed to set up queue and exchange: %v", err)
		os.Exit(1)
	}
}

func setupQueueAndExchange() error {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return err
	}
	channel, err := conn.Channel()
	if err != nil {
		return err
	}

	err = channel.ExchangeDeclare(amqpExchangeName, amqpExchangeType, true, false, false, false, nil)
	if err != nil {
		return err
	}

	q, err := channel.QueueDeclare(amqpQueue, true, false, false, false, nil)
	if err != nil {
		return err
	}

	err = channel.QueueBind(q.Name, "", amqpExchangeName, false, nil)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	name := "amqp"
	version := "1.0.0"

	service, err := patron.New(name, version, patron.TextLogger())
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
	err = routeSNSTopicToSQSQueue(snsAPI, sqsQueueURL, snsTopicArn)
	if err != nil {
		log.Fatalf("failed to route sns to sqs: %v", err)
	}

	// Create an SNS publisher
	snsPub, err := patronsns.New(snsAPI)
	if err != nil {
		log.Fatalf("failed to create sns publisher: %v", err)
	}

	// Create an SQS publisher
	sqsPub, err := patronsqs.New(sqsAPI)
	if err != nil {
		log.Fatalf("failed to create sqs publisher: %v", err)
	}

	// Initialise the AMQP component
	amqpCmp, err := newAmqpComponent(amqpURL, amqpQueue, snsTopicArn, snsPub, sqsPub, sqsQueueURL)
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

	// 15 attempts 1 seconds separated to retrieve valid session
	var s *session.Session = nil
	var err error = nil
	for i := 0; i < 15; i++ {
		s, err = session.NewSession(
			&aws.Config{
				Region:      aws.String(awsRegion),
				Credentials: credentials.NewStaticCredentials(awsID, awsSecret, awsToken),
			},
			&aws.Config{Endpoint: aws.String(endpoint)},
		)
		if err == nil {
			return s
		}
		time.Sleep(1 * time.Second)
	}
	// this will panic if error is not null
	return session.Must(s, err)
}

func createSQSQueue(api sqsiface.SQSAPI) (string, error) {
	out, err := api.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(awsSQSQueue),
	})
	if err != nil {
		return "", err
	}
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

func routeSNSTopicToSQSQueue(snsAPI snsiface.SNSAPI, sqsQueueArn, topicArn string) error {
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

func newAmqpComponent(url, queue, snsTopicArn string, snsPub patronsns.Publisher, sqsPub patronsqs.Publisher,
	sqsQueueURL string) (*amqpComponent, error) {
	amqpCmp := amqpComponent{
		snsTopicArn: snsTopicArn,
		snsPub:      snsPub,
		sqsPub:      sqsPub,
		sqsQueueURL: sqsQueueURL,
	}

	cmp, err := patronamqp.New(url, queue, amqpCmp.Process, patronamqp.Retry(10, 1*time.Second))
	if err != nil {
		return nil, err
	}

	amqpCmp.cmp = cmp

	return &amqpCmp, nil
}

func (ac *amqpComponent) Process(ctx context.Context, batch patronamqp.Batch) {
	for _, msg := range batch.Messages() {
		if ctx.Err() != nil {
			log.FromContext(ctx).Info("context cancelled, exiting process function")
		}
		logger := log.FromContext(msg.Context())

		var u examples.User

		err := protobuf.DecodeRaw(msg.Body(), &u)
		if err != nil {
			logger.Errorf("failed to decode message: %v", err)
			err = msg.NACK()
			if err != nil {
				logger.Errorf("failed to NACK message: %v", err)
			}
		}

		payload, err := json.Encode(u)
		if err != nil {
			logger.Errorf("failed to encode message: %v", err)
			err = msg.NACK()
			if err != nil {
				logger.Errorf("failed to NACK message: %v", err)
			}
		}

		input := &sns.PublishInput{
			Message:   aws.String(string(payload)),
			TargetArn: aws.String(ac.snsTopicArn),
		}
		_, err = ac.snsPub.Publish(msg.Context(), input)
		if err != nil {
			logger.Errorf("failed to publish message to SNS: %v", err)
			err = msg.NACK()
			if err != nil {
				logger.Errorf("failed to NACK message: %v", err)
			}
		}

		sqsMsg := &sqs.SendMessageInput{
			MessageBody: aws.String(string(payload)),
			QueueUrl:    aws.String(ac.sqsQueueURL),
		}

		_, err = ac.sqsPub.Publish(msg.Context(), sqsMsg)
		if err != nil {
			logger.Errorf("failed to publish message to SQS: %v", err)
			err = msg.NACK()
			if err != nil {
				logger.Errorf("failed to NACK message: %v", err)
			}
		}

		logger.Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	}
}
