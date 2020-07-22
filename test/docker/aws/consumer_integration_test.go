// +build integration

package aws

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	patronSQS "github.com/beatlabs/patron/client/sqs"
	sqsConsumer "github.com/beatlabs/patron/component/async/sqs"
	"github.com/beatlabs/patron/correlation"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type message struct {
	ID string `json:"id"`
}

func Test_SQS_Consume(t *testing.T) {
	const queueName = "test-sqs-consume"
	const correlationID = "123"

	api, err := createSQSAPI(runtime.getSQSEndpoint())
	require.NoError(t, err)
	queue, err := createSQSQueue(api, queueName)
	require.NoError(t, err)

	sent := sendMessage(t, api, correlationID, queue, "1", "2", "3")

	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)

	factory, err := sqsConsumer.NewFactory(api, queueName)
	require.NoError(t, err)
	cns, err := factory.Create()
	require.NoError(t, err)
	ch, chErr, err := cns.Consume(context.Background())
	require.NoError(t, err)

	count := 0

	chReceived := make(chan []*message)

	go func() {
		received := make([]*message, 0, len(sent))

		for {
			select {
			case msg := <-ch:
				var got message
				require.NoError(t, msg.Decode(&got))
				received = append(received, &got)
				require.NoError(t, msg.Ack())
				count++
				if count == len(sent) {
					chReceived <- received
					return
				}
			case err := <-chErr:
				require.NoError(t, err)
				return
			}
		}
	}()

	assert.Equal(t, sent, <-chReceived)
	assert.Len(t, mtr.FinishedSpans(), 3)

	for _, span := range mtr.FinishedSpans() {
		expected := map[string]interface{}{
			"component":     "sqs-consumer",
			"error":         false,
			"span.kind":     ext.SpanKindEnum("consumer"),
			"version":       "dev",
			"correlationID": correlationID,
		}
		assert.Equal(t, expected, span.Tags())
	}
}

func sendMessage(t *testing.T, api sqsiface.SQSAPI, correlationID, queue string, ids ...string) []*message {
	pub, err := patronSQS.NewPublisher(api)
	require.NoError(t, err)

	ctx := correlation.ContextWithID(context.Background(), correlationID)

	sentMessages := make([]*message, 0, len(ids))

	for _, id := range ids {
		sentMsg := &message{
			ID: id,
		}
		sentMsgBody, err := json.Marshal(sentMsg)
		require.NoError(t, err)

		msg, err := patronSQS.NewMessageBuilder().
			QueueURL(queue).
			Body(string(sentMsgBody)).
			WithDelaySeconds(1).Body(string(sentMsgBody)).Build()
		require.NoError(t, err)

		msgID, err := pub.Publish(ctx, *msg)
		assert.NoError(t, err)
		assert.NotEmpty(t, msgID)

		sentMessages = append(sentMessages, sentMsg)
	}

	return sentMessages
}
