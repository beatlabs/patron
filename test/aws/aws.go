package aws

import (
	"context"
	"fmt"

	awsV2 "github.com/aws/aws-sdk-go-v2/aws"
	awsV2Config "github.com/aws/aws-sdk-go-v2/config"
	awsV2Credentials "github.com/aws/aws-sdk-go-v2/credentials"
	awsV2Sns "github.com/aws/aws-sdk-go-v2/service/sns"
	awsV2Sqs "github.com/aws/aws-sdk-go-v2/service/sqs"
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

// CreateSNSAPIV2 helper function.
func CreateSNSAPIV2(region, endpoint string) (*awsV2Sns.Client, error) {
	cfg, err := createConfigV2(awsV2Sns.ServiceID, region, endpoint)
	if err != nil {
		return nil, err
	}

	api := awsV2Sns.NewFromConfig(cfg)

	return api, nil
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

// CreateSNSTopicV2 helper function.
func CreateSNSTopicV2(api *awsV2Sns.Client, topic string) (string, error) {
	out, err := api.CreateTopic(context.Background(), &awsV2Sns.CreateTopicInput{
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

// CreateSQSAPIV2 helper function.
func CreateSQSAPIV2(region, endpoint string) (*awsV2Sqs.Client, error) {
	cfg, err := createConfigV2(awsV2Sqs.ServiceID, region, endpoint)
	if err != nil {
		return nil, err
	}

	api := awsV2Sqs.NewFromConfig(cfg)

	return api, nil
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

// CreateSQSQueueV2 helper function.
func CreateSQSQueueV2(api *awsV2Sqs.Client, queueName string) (string, error) {
	out, err := api.CreateQueue(context.Background(), &awsV2Sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", err
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

func createConfigV2(awsServiceID, awsRegion, awsEndpoint string) (awsV2.Config, error) {
	customResolver := awsV2.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (awsV2.Endpoint, error) {
		if service == awsServiceID && region == awsRegion {
			return awsV2.Endpoint{
				URL:           awsEndpoint,
				SigningRegion: awsRegion,
			}, nil
		}
		// returning EndpointNotFoundError will allow the service to fallback to it's default resolution
		return awsV2.Endpoint{}, &awsV2.EndpointNotFoundError{}
	})

	cfg, err := awsV2Config.LoadDefaultConfig(context.TODO(),
		awsV2Config.WithRegion(awsRegion),
		awsV2Config.WithEndpointResolverWithOptions(customResolver),
		awsV2Config.WithCredentialsProvider(awsV2.NewCredentialsCache(awsV2Credentials.NewStaticCredentialsProvider("test", "test", ""))),
	)
	if err != nil {
		return awsV2.Config{}, fmt.Errorf("failed to create AWS config: %w", err)
	}

	return cfg, nil
}
