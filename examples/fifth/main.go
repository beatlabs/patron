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
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/async"
	patronsqs "github.com/beatlabs/patron/async/sqs"
	"github.com/beatlabs/patron/log"
)

const (
	awsRegion      = "eu-west-1"
	awsID          = "test"
	awsSecret      = "test"
	awsToken       = "token"
	awsSQSEndpoint = "http://localhost:4576"
	awsSQSQueue    = "patron"
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
	sqsAPI := sqs.New(
		session.Must(
			session.NewSession(
				&aws.Config{
					Region:      aws.String(awsRegion),
					Credentials: credentials.NewStaticCredentials(awsID, awsSecret, awsToken),
				},
				&aws.Config{Endpoint: aws.String(awsSQSEndpoint)},
			),
		),
	)
	sqsCmp, err := createSQSComponent(sqsAPI)
	if err != nil {
		log.Fatalf("failed to create sqs component: %v", err)
	}

	// Run the server
	srv, err := patron.New(name, version, patron.Components(sqsCmp.cmp))
	if err != nil {
		log.Fatalf("failed to create service: %v", err)
	}

	ctx := context.Background()
	err = srv.Run(ctx)
	if err != nil {
		log.Fatalf("failed to run service: %v", err)
	}
}

type sqsComponent struct {
	cmp patron.Component
}

func createSQSComponent(api sqsiface.SQSAPI) (*sqsComponent, error) {
	sqsCmp := sqsComponent{}

	cf, err := patronsqs.NewFactory(api, awsSQSQueue)
	if err != nil {
		return nil, err
	}

	cmp, err := async.New("sqs-cmp", cf, sqsCmp.Process).
		WithRetries(10).
		WithRetryWait(10 * time.Second).
		Create()
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

	log.FromContext(msg.Context()).Infof("request processed: %v", got.Message)
	return nil
}
