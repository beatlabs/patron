//go:build integration

package sqs

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewFromConfig(t *testing.T) {
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	awsRegion := "eu-west-1"

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(awsRegion),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider("test", "test", "token"))),
	)
	require.NoError(t, err)

	client := NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String("http://localhost:4566")
		o.Region = awsRegion
	})

	// Add your assertions here to test the behavior of the client

	assert.NotNil(t, client)

	out, err := client.CreateQueue(context.Background(), &sqs.CreateQueueInput{
		QueueName: aws.String("test-queue"),
	})

	require.NoError(t, err)

	assert.NotEmpty(t, out.QueueUrl)
	require.NoError(t, tracePublisher.ForceFlush(context.Background()))

	assert.Len(t, exp.GetSpans(), 1)
}
