package sns

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewFromConfig(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tracePublisher, err := patrontrace.Setup("test", nil, exp)
	require.NoError(t, err)

	awsRegion := "eu-west-1"

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == sns.ServiceID && region == awsRegion {
			return aws.Endpoint{
				URL:           "http://localhost:4566",
				SigningRegion: awsRegion,
			}, nil
		}
		// returning EndpointNotFoundError will allow the service to fallback to it's default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(awsRegion),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider("test", "test", "token"))),
	)
	require.NoError(t, err)

	client := NewFromConfig(cfg)

	// Add your assertions here to test the behavior of the client

	assert.NotNil(t, client)

	out, err := client.CreateTopic(context.Background(), &sns.CreateTopicInput{
		Name: aws.String("test-topic"),
	})

	assert.NoError(t, err)

	assert.NotEmpty(t, out.TopicArn)
	assert.NoError(t, tracePublisher.ForceFlush(context.Background()))

	assert.Len(t, exp.GetSpans(), 1)
}
