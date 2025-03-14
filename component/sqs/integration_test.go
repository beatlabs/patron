//go:build integration

package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	region   = "eu-west-1"
	endpoint = "http://localhost:4566"
)

type testMessage struct {
	ID string `json:"id"`
}

func Test_SQS_Consume(t *testing.T) {
	// Trace setup
	t.Cleanup(func() { traceExporter.Reset() })
	// Metrics setup
	ctx := context.Background()

	shutdownProvider, collectMetrics := test.SetupMetrics(ctx, t)
	defer shutdownProvider()

	const queueName = "test-sqs-consume"
	const correlationID = "123"

	api, err := createSQSAPI(region, endpoint)
	require.NoError(t, err)
	queue, err := createSQSQueue(api, queueName)
	require.NoError(t, err)

	sent := sendMessage(t, api, correlationID, queue, "1", "2", "3")
	traceExporter.Reset()

	chReceived := make(chan []*testMessage)
	received := make([]*testMessage, 0)
	count := 0

	procFunc := func(ctx context.Context, b Batch) {
		if ctx.Err() != nil {
			return
		}

		for _, msg := range b.Messages() {
			var m1 testMessage
			require.NoError(t, json.Unmarshal(msg.Body(), &m1))
			received = append(received, &m1)
			require.NoError(t, msg.ACK())
		}

		count += len(b.Messages())
		if count == len(sent) {
			chReceived <- received
		}
	}

	cmp, err := New("123", queueName, api, procFunc, WithMaxMessages(10),
		WithPollWaitSeconds(20), WithVisibilityTimeout(30), WithQueueStatsInterval(10*time.Millisecond))
	require.NoError(t, err)

	go func() { assert.NoError(t, cmp.Run(context.Background())) }()

	got := <-chReceived

	assert.ElementsMatch(t, sent, got)

	require.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expected := createStubSpan("sqs-consumer", "")

	spans := traceExporter.GetSpans()

	assert.Len(t, got, 3)
	test.AssertSpan(t, expected, spans[0])
	test.AssertSpan(t, expected, spans[1])
	test.AssertSpan(t, expected, spans[2])

	// Metrics
	collectedMetrics := collectMetrics(3)
	test.AssertMetric(t, collectedMetrics.ScopeMetrics[0].Metrics, "sqs.message.age")
	test.AssertMetric(t, collectedMetrics.ScopeMetrics[0].Metrics, "sqs.message.counter")
	test.AssertMetric(t, collectedMetrics.ScopeMetrics[0].Metrics, "sqs.queue.size")
}

func sendMessage(t *testing.T, client *sqs.Client, correlationID, queue string, ids ...string) []*testMessage {
	ctx := correlation.ContextWithID(context.Background(), correlationID)

	sentMessages := make([]*testMessage, 0, len(ids))

	for _, id := range ids {
		sentMsg := &testMessage{
			ID: id,
		}
		sentMsgBody, err := json.Marshal(sentMsg)
		require.NoError(t, err)

		msg := &sqs.SendMessageInput{
			DelaySeconds: int32(1),
			MessageBody:  aws.String(string(sentMsgBody)),
			QueueUrl:     aws.String(queue),
		}

		msgID, err := client.SendMessage(ctx, msg)
		require.NoError(t, err)
		assert.NotEmpty(t, msgID)

		sentMessages = append(sentMessages, sentMsg)
	}

	return sentMessages
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
	cfg, err := awsConfig.LoadDefaultConfig(context.Background(),
		awsConfig.WithRegion(region),
		awsConfig.WithCredentialsProvider(aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("test", "test", ""))),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	api := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.Region = region
	})

	return api, nil
}
