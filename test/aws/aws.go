package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// CreateSNSAPI helper function.
func CreateSNSAPI(region, endpoint string) (snsiface.SNSAPI, error) {
	ses, err := createSession(region, endpoint)
	if err != nil {
		return nil, err
	}

	cfg := &aws.Config{
		Region: aws.String(region),
	}

	return sns.New(ses, cfg), nil
}

// CreateSNSTopic helper function.
func CreateSNSTopic(api snsiface.SNSAPI, topic string) (string, error) {
	out, err := api.CreateTopic(&sns.CreateTopicInput{
		Name: aws.String(topic),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create topic %s: %w", topic, err)
	}

	return *out.TopicArn, nil
}

// CreateSQSAPI helper function.
func CreateSQSAPI(region, endpoint string) (sqsiface.SQSAPI, error) {
	ses, err := createSession(region, endpoint)
	if err != nil {
		return nil, err
	}

	cfg := &aws.Config{
		Region: aws.String(region),
	}

	return sqs.New(ses, cfg), nil
}

// CreateSQSQueue helper function.
func CreateSQSQueue(api sqsiface.SQSAPI, queueName string) (string, error) {
	out, err := api.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create SQS queue %s: %w", queueName, err)
	}
	return *out.QueueUrl, nil
}

func createSession(region, endpoint string) (*session.Session, error) {
	ses, err := session.NewSession(
		aws.NewConfig().
			WithEndpoint(endpoint).
			WithRegion(region).
			WithCredentials(credentials.NewStaticCredentials("test", "test", "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS endpoint: %w", err)
	}

	return ses, nil
}
