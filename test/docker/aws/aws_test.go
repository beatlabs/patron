// +build integration

package aws

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"

	"github.com/aws/aws-sdk-go/service/sqs"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"

	patronDocker "github.com/beatlabs/patron/test/docker"
	"github.com/ory/dockertest"
)

const (
	testSnsRegion    = "eu-west-1"
	testSNSTopic     = "test-topic"
	testSQSQueueName = "test-publish-message"
)

var (
	runtime *awsRuntime
)

func TestMain(m *testing.M) {
	var err error
	runtime, err = create(60 * time.Second)
	if err != nil {
		fmt.Printf("could not create AWS runtime: %v\n", err)
		os.Exit(1)
	}

	exitCode := m.Run()

	ee := runtime.Teardown()
	if len(ee) > 0 {
		for _, err := range ee {
			fmt.Printf("could not tear down containers: %v\n", err)
		}
	}

	os.Exit(exitCode)
}

type awsRuntime struct {
	patronDocker.Runtime
}

func create(expiration time.Duration) (*awsRuntime, error) {
	br, err := patronDocker.NewRuntime(expiration)
	if err != nil {
		return nil, fmt.Errorf("could not create base runtime: %w", err)
	}

	runtime := &awsRuntime{Runtime: *br}

	runOptions := &dockertest.RunOptions{
		Repository: "localstack/localstack",
		Tag:        "0.11.2",
		Env: []string{
			"LOCALSTACK_SERVICES=sqs,sns",
			"LOCALSTACK_DEBUG=1",
			"LOCALSTACK_DATA_DIR=/tmp/localstack/data",
			"AWS_ACCESS_KEY_ID=test",
			"AWS_SECRET_ACCESS_KEY=test",
			"AWS_DEFAULT_REGION=eu-west-1",
		}}
	_, err = runtime.RunWithOptions(runOptions)
	if err != nil {
		return nil, fmt.Errorf("could not start mysql: %w", err)
	}

	// wait until the container is ready
	err = runtime.Pool().Retry(func() error {
		snsAPI, err := createSNSAPI(runtime.getSNSEndpoint())
		if err != nil {
			return err
		}

		_, err = createSNSTopic(snsAPI, testSNSTopic)
		if err != nil {
			return err
		}

		sqsAPI, err := createSQSAPI(runtime.getSQSEndpoint())
		if err != nil {
			return err
		}

		_, err = createSQSQueue(sqsAPI, testSQSQueueName)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		for _, err1 := range runtime.Teardown() {
			fmt.Printf("failed to teardown: %v\n", err1)
		}
		return nil, fmt.Errorf("container not ready: %w", err)
	}

	return runtime, nil
}

func (s *awsRuntime) getSNSEndpoint() string {
	return fmt.Sprintf("http://localhost:%s", s.Resources()[0].GetPort("4575/tcp"))
}

func (s *awsRuntime) getSQSEndpoint() string {
	return fmt.Sprintf("http://localhost:%s", s.Resources()[0].GetPort("4576/tcp"))
}

func createSNSAPI(endpoint string) (snsiface.SNSAPI, error) {
	sess, err := session.NewSession(
		aws.NewConfig().
			WithEndpoint(endpoint).
			WithRegion(testSnsRegion).
			WithCredentials(credentials.NewStaticCredentials("test", "test", "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS endpoint: %w", err)
	}

	cfg := &aws.Config{
		Region: aws.String(testSnsRegion),
	}

	return sns.New(sess, cfg), nil
}

func createSNSTopic(api snsiface.SNSAPI, topic string) (string, error) {
	out, err := api.CreateTopic(&sns.CreateTopicInput{
		Name: aws.String(topic),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create topic %s: %w", topic, err)
	}

	return *out.TopicArn, nil
}

func createSQSAPI(endpoint string) (sqsiface.SQSAPI, error) {
	sess, err := session.NewSession(
		aws.NewConfig().
			WithEndpoint(endpoint).
			WithRegion(testSnsRegion).
			WithCredentials(credentials.NewStaticCredentials("test", "test", "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS endpoint: %w", err)
	}

	cfg := &aws.Config{
		Region: aws.String(testSnsRegion),
	}

	return sqs.New(sess, cfg), nil
}

func createSQSQueue(api sqsiface.SQSAPI, queueName string) (string, error) {
	out, err := api.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create SQS queue %s: %w", queueName, err)
	}
	return *out.QueueUrl, nil
}
