//go:build integration
// +build integration

package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
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

type sampleMsg struct {
	Foo string `json:"foo"`
	Bar string `json:"bar"`
}

func Test_SQS_Publish_Message(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	t.Cleanup(func() { mtr.Reset() })

	const queueName = "test-sqs-publish-v2"

	api, err := createSQSAPI(region, endpoint)
	require.NoError(t, err)
	queue, err := createSQSQueue(api, queueName)
	require.NoError(t, err)

	pub, err := New(api)
	require.NoError(t, err)

	sentMsg := &sampleMsg{
		Foo: "foo",
		Bar: "bar",
	}
	sentMsgBody, err := json.Marshal(sentMsg)
	require.NoError(t, err)

	msg := &sqs.SendMessageInput{
		MessageBody: aws.String(string(sentMsgBody)),
		QueueUrl:    aws.String(queue),
	}

	msgID, err := pub.Publish(context.Background(), msg)
	assert.NoError(t, err)
	assert.IsType(t, "string", msgID)

	out, err := api.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
		QueueUrl:        &queue,
		WaitTimeSeconds: int32(2),
	})
	require.NoError(t, err)
	assert.Len(t, out.Messages, 1)
	assert.Equal(t, string(sentMsgBody), *out.Messages[0].Body)

	expected := map[string]interface{}{
		"component": "sqs-publisher",
		"error":     false,
		"span.kind": ext.SpanKindEnum("producer"),
		"version":   "dev",
	}
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
	assert.Equal(t, 1, testutil.CollectAndCount(publishDurationMetrics, "client_sqs_publish_duration_seconds"))
}

func createSQSQueue(api SQSAPI, queueName string) (string, error) {
	out, err := api.CreateQueue(context.Background(), &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", err
	}

	return *out.QueueUrl, nil
}

type SQSAPI interface {
	CreateQueue(ctx context.Context, params *sqs.CreateQueueInput, optFns ...func(*sqs.Options)) (*sqs.CreateQueueOutput, error)
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
}

func createSQSAPI(region, endpoint string) (*sqs.Client, error) {
	cfg, err := createConfig(sqs.ServiceID, region, endpoint)
	if err != nil {
		return nil, err
	}

	api := sqs.NewFromConfig(cfg)

	return api, nil
}

func createConfig(awsServiceID, awsRegion, awsEndpoint string) (aws.Config, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, _ ...interface{}) (aws.Endpoint, error) {
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
