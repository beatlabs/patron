// +build integration

package aws

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	patronsqsclient "github.com/beatlabs/patron/client/sqs/v2"
	patronsqs "github.com/beatlabs/patron/component/sqs"
	"github.com/beatlabs/patron/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type message struct {
	ID string `json:"id"`
}

func Test_SQS_Consume(t *testing.T) {
	defer mtr.Reset()

	const queueName = "test-sqs-consume"
	const correlationID = "123"

	api, err := createSQSAPI(runtime.getSQSEndpoint())
	require.NoError(t, err)
	queue, err := createSQSQueue(api, queueName)
	require.NoError(t, err)

	sent := sendMessage(t, api, correlationID, queue, "1", "2", "3")
	mtr.Reset()

	chReceived := make(chan []*message)
	received := make([]*message, 0)
	count := 0

	procFunc := func(ctx context.Context, b patronsqs.Batch) {
		if ctx.Err() != nil {
			return
		}

		for _, msg := range b.Messages() {
			var m1 message
			require.NoError(t, json.Unmarshal(msg.Body(), &m1))
			received = append(received, &m1)
			require.NoError(t, msg.ACK())
		}

		count += len(b.Messages())
		if count == len(sent) {
			chReceived <- received
		}
	}

	cmp, err := patronsqs.New("123", queueName, api, procFunc, patronsqs.MaxMessages(10),
		patronsqs.PollWaitSeconds(20), patronsqs.VisibilityTimeout(30))
	require.NoError(t, err)

	go func() { require.NoError(t, cmp.Run(context.Background())) }()

	got := <-chReceived

	assert.ElementsMatch(t, sent, got)
	assert.Len(t, mtr.FinishedSpans(), 3)
}

func sendMessage(t *testing.T, api sqsiface.SQSAPI, correlationID, queue string, ids ...string) []*message {
	pub, err := patronsqsclient.New(api)
	require.NoError(t, err)

	ctx := correlation.ContextWithID(context.Background(), correlationID)

	sentMessages := make([]*message, 0, len(ids))

	for _, id := range ids {
		sentMsg := &message{
			ID: id,
		}
		sentMsgBody, err := json.Marshal(sentMsg)
		require.NoError(t, err)

		msg := &sqs.SendMessageInput{
			DelaySeconds: aws.Int64(1),
			MessageBody:  aws.String(string(sentMsgBody)),
			QueueUrl:     aws.String(queue),
		}

		msgID, err := pub.Publish(ctx, msg)
		assert.NoError(t, err)
		assert.NotEmpty(t, msgID)

		sentMessages = append(sentMessages, sentMsg)
	}

	return sentMessages
}
