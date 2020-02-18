// +build integration

package sqs

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// These values are taken from examples/docker-compose.yml
	testSqsEndpoint string = "http://localhost:4576"
	testSqsRegion   string = "eu-west-1"
)

func Test_Publish_Message_FIFO_Queue(t *testing.T) {
	api := createAPI(t)
	pub := createPublisher(t, api)
	queueURL := createQueue(t, api)
	fifoQueueURL := createFifoQueue(t, api)

	msg := createMsg(t, queueURL)
	fifoMsg := createFIFOMsg(t, fifoQueueURL)

	msgID, err := pub.Publish(context.Background(), msg)
	assert.NoError(t, err)
	assert.IsType(t, "string", msgID)

	fifoMsgID, err := pub.Publish(context.Background(), fifoMsg)
	assert.NoError(t, err)
	assert.IsType(t, "string", fifoMsgID)
}

func createAPI(t *testing.T) sqsiface.SQSAPI {
	sess, err := session.NewSession(
		aws.NewConfig().
			WithEndpoint(testSqsEndpoint).
			WithRegion(testSqsRegion),
	)
	require.NoError(t, err)

	cfg := &aws.Config{
		Region: aws.String(testSqsRegion),
	}

	return sqs.New(sess, cfg)
}

func createPublisher(t *testing.T, api sqsiface.SQSAPI) Publisher {
	p, err := NewPublisher(api)
	require.NoError(t, err)

	return p
}

func createQueue(t *testing.T, api sqsiface.SQSAPI) (topicArn string) {
	out, err := api.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String("test-queue"),
	})
	require.NoError(t, err)

	return *out.QueueUrl
}

func createFifoQueue(t *testing.T, api sqsiface.SQSAPI) (topicArn string) {
	input := &sqs.CreateQueueInput{
		QueueName: aws.String("test-queue.fifo"),
	}
	input.SetAttributes(map[string]*string{
		"FifoQueue":                 aws.String("true"),
		"ContentBasedDeduplication": aws.String("true"),
	})
	out, err := api.CreateQueue(input)
	require.NoError(t, err)

	return *out.QueueUrl
}

func createMsg(t *testing.T, queueURL string) Message {
	b := NewMessageBuilder()

	msg, err := b.
		Body("test msg").
		QueueURL(queueURL).
		WithStringAttribute("foo", "bar").
		WithNumberAttribute("bar", 1337).
		WithBinaryAttribute("baz", []byte("baz")).
		WithDelaySeconds(1).
		Build()
	require.NoError(t, err)

	return *msg
}

func createFIFOMsg(t *testing.T, queueURL string) Message {
	b := NewMessageBuilder()

	msg, err := b.
		Body("test msg").
		QueueURL(queueURL).
		WithStringAttribute("foo", "bar").
		WithNumberAttribute("bar", 1337).
		WithBinaryAttribute("baz", []byte("baz")).
		WithDeduplicationID("dedup-id").
		WithGroupID("group-id").
		Build()
	require.NoError(t, err)

	return *msg
}
