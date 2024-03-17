//go:build integration
// +build integration

package sns

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	region   = "eu-west-1"
	endpoint = "http://localhost:4566"
)

func Test_SNS_Publish_Message_v2(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	t.Cleanup(func() { mtr.Reset() })

	const topic = "test_publish_message_v2"
	api, err := createSNSAPI(region, endpoint)
	require.NoError(t, err)
	arn, err := createSNSTopic(api, topic)
	require.NoError(t, err)
	pub, err := New(api)
	require.NoError(t, err)
	input := &sns.PublishInput{
		Message:  aws.String(topic),
		TopicArn: aws.String(arn),
	}

	msgID, err := pub.Publish(context.Background(), input)
	assert.NoError(t, err)
	assert.IsType(t, "string", msgID)
	expected := map[string]interface{}{
		"component": "sns-publisher",
		"error":     false,
		"span.kind": ext.SpanKindEnum("producer"),
		"version":   "dev",
	}
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
	// Metrics
	assert.Equal(t, 1, testutil.CollectAndCount(publishDurationMetrics, "client_sns_publish_duration_seconds"))
}

func createSNSAPI(region, endpoint string) (*sns.Client, error) {
	cfg, err := createConfig(sns.ServiceID, region, endpoint)
	if err != nil {
		return nil, err
	}

	api := sns.NewFromConfig(cfg)

	return api, nil
}

func createSNSTopic(api SNSAPI, topic string) (string, error) {
	out, err := api.CreateTopic(context.Background(), &sns.CreateTopicInput{
		Name: aws.String(topic),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create topic %s: %w", topic, err)
	}

	return *out.TopicArn, nil
}

type SNSAPI interface {
	CreateTopic(ctx context.Context, params *sns.CreateTopicInput, optFns ...func(*sns.Options)) (*sns.CreateTopicOutput, error)
}

func createConfig(awsServiceID, awsRegion, awsEndpoint string) (aws.Config, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == awsServiceID && region == awsRegion {
			return aws.Endpoint{
				URL:           awsEndpoint,
				SigningRegion: awsRegion,
			}, nil
		}
		// returning EndpointNotFoundError will allow the service to fallback to it's default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("test", "test", ""))),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to create AWS config: %w", err)
	}

	return cfg, nil
}
