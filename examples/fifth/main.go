package main

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/async"
	patronsqs "github.com/beatlabs/patron/async/sqs"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
)

const (
	awsRegion   = "eu-west-1"
	awsID       = "test"
	awsSecret   = "test"
	awsToken    = "token"
	awsEndpoint = "http://localhost:4576"
	awsQueue    = "patron"
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

	sqsCmp, err := newSQSComponent()
	if err != nil {
		log.Fatalf("failed to create sqs component: %v", err)
	}

	srv, err := patron.New(name, version, patron.Components(sqsCmp.cmp))
	if err != nil {
		log.Fatalf("failed to create service: %v", err)
	}

	err = srv.Run()
	if err != nil {
		log.Fatalf("failed to run service: %v", err)
	}
}

type sqsComponent struct {
	cmp patron.Component
}

func newSQSComponent() (*sqsComponent, error) {

	sqsCmp := sqsComponent{}

	ses, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsID, awsSecret, awsToken),
		Endpoint:    aws.String(awsEndpoint),
	})
	if err != nil {
		return nil, err
	}

	cf, err := patronsqs.NewFactory(sqs.New(ses), awsQueue)
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
	var u examples.User

	err := msg.Decode(&u)
	if err != nil {
		return err
	}

	log.FromContext(msg.Context()).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return nil
}
