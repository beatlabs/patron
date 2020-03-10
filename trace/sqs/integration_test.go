// +build integration

package sqs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
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

type sampleMsg struct {
	Foo string `json:"foo"`
	Bar string `json:"bar"`
}

func Test_Publish_Message(t *testing.T) {
	api := createAPI(t)
	pub := createPublisher(t, api)

	stdQueueName := "test-publish-message"
	fifoQueueName := "test-publish-message.fifo"

	deleteQueue(t, api, stdQueueName)
	deleteQueue(t, api, fifoQueueName)

	stdQueueURL := createQueue(t, api, stdQueueName)
	fifoQueueURL := createFIFOQueue(t, api, fifoQueueName)

	sentMsg := &sampleMsg{
		Foo: "foo",
		Bar: "bar",
	}
	sentMsgBody, err := json.Marshal(sentMsg)
	require.NoError(t, err)

	testCases := map[string]struct {
		queueURL                string
		preconfiguredMsgBuilder *MessageBuilder
	}{
		"normal queue": {
			queueURL: stdQueueURL,
			preconfiguredMsgBuilder: NewMessageBuilder().
				QueueURL(stdQueueURL).
				Body(string(sentMsgBody)).
				WithDelaySeconds(1),
		},
		"FIFO queue": {
			queueURL: fifoQueueURL,
			preconfiguredMsgBuilder: NewMessageBuilder().
				QueueURL(fifoQueueURL).
				Body(string(sentMsgBody)).
				WithGroupID("group-id"),
		},
	}
	for name, tC := range testCases {
		t.Run(name, func(t *testing.T) {
			msg, err := tC.preconfiguredMsgBuilder.Body(string(sentMsgBody)).Build()
			require.NoError(t, err)

			msgID, err := pub.Publish(context.Background(), *msg)
			assert.NoError(t, err)
			assert.IsType(t, "string", msgID)

			out, err := api.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl:        &tC.queueURL,
				WaitTimeSeconds: aws.Int64(2),
			})
			require.NoError(t, err)
			assert.Len(t, out.Messages, 1)
			assert.Equal(t, string(sentMsgBody), *out.Messages[0].Body)
		})
	}
}

func createAPI(t *testing.T) sqsiface.SQSAPI {

	sess, err := session.NewSession(
		aws.NewConfig().
			WithEndpoint(testSqsEndpoint).
			WithRegion(testSqsRegion).
			WithCredentials(credentials.NewStaticCredentials("test", "test", "")),
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

func createQueue(t *testing.T, api sqsiface.SQSAPI, queueName string) string {
	out, err := api.CreateQueue(&sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	})
	require.NoError(t, err)
	return *out.QueueUrl
}

func createFIFOQueue(t *testing.T, api sqsiface.SQSAPI, queueName string) string {
	input := &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	}
	input.SetAttributes(map[string]*string{
		"FifoQueue":                 aws.String("true"),
		"ContentBasedDeduplication": aws.String("true"),
	})
	out, err := api.CreateQueue(input)
	require.NoError(t, err)

	return *out.QueueUrl
}

func deleteQueue(t *testing.T, api sqsiface.SQSAPI, queueName string) {
	out, err := api.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() != sqs.ErrCodeQueueDoesNotExist {
			t.Fatalf("unexpected error received: %v", err)
		}
	}

	if err != nil {
		return
	}

	queueURL := *out.QueueUrl

	_, err = api.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(queueURL),
	})
	require.NoError(t, err)
}
