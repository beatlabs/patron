package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/beatlabs/patron"
	patrongrpc "github.com/beatlabs/patron/client/grpc"
	"github.com/beatlabs/patron/component/async"
	patronsqs "github.com/beatlabs/patron/component/async/sqs"
	"github.com/beatlabs/patron/component/grpc/greeter"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	"google.golang.org/grpc"
)

const (
	awsRegion      = "eu-west-1"
	awsID          = "test"
	awsSecret      = "test"
	awsToken       = "token"
	awsSQSEndpoint = "http://localhost:4566"
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

	service, err := patron.New(name, version)
	if err != nil {
		fmt.Printf("failed to set up service: %v", err)
		os.Exit(1)
	}

	cc, err := patrongrpc.Dial("localhost:50006", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to dial grpc connection: %v", err)
	}
	defer func() {
		_ = cc.Close()
	}()

	greeter := greeter.NewGreeterClient(cc)

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
	sqsCmp, err := createSQSComponent(sqsAPI, greeter)
	if err != nil {
		log.Fatalf("failed to create sqs component: %v", err)
	}

	// Run the server
	ctx := context.Background()
	err = service.WithComponents(sqsCmp.cmp).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service: %v", err)
	}
}

type sqsComponent struct {
	cmp     patron.Component
	greeter greeter.GreeterClient
}

func createSQSComponent(api sqsiface.SQSAPI, greeter greeter.GreeterClient) (*sqsComponent, error) {
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
	sqsCmp.greeter = greeter

	return &sqsCmp, nil
}

func (ac *sqsComponent) Process(msg async.Message) error {
	var u examples.User

	err := msg.Decode(&u)
	if err != nil {
		return err
	}

	logger := log.FromContext(msg.Context())
	logger.Infof("request processed: %v, sending request to sixth service", u.String())

	reply, err := ac.greeter.SayHello(msg.Context(), &greeter.HelloRequest{Firstname: u.GetFirstname(), Lastname: u.GetLastname()})
	if err != nil {
		logger.Errorf("failed to send request: %v", err)
	}

	logger.Infof("Reply from sixth service: %s", reply.GetMessage())
	return nil
}
