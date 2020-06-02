// +build integration

package aws

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/opentracing/opentracing-go/ext"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	patronSQS "github.com/beatlabs/patron/client/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sampleMsg struct {
	Foo string `json:"foo"`
	Bar string `json:"bar"`
}

func Test_SQS_Publish_Message(t *testing.T) {
	const queueName = "test-sqs-publish"
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)

	api, err := createSQSAPI(runtime.getSQSEndpoint())
	require.NoError(t, err)
	queue, err := createSQSQueue(api, queueName)
	require.NoError(t, err)

	pub, err := patronSQS.NewPublisher(api)
	require.NoError(t, err)

	sentMsg := &sampleMsg{
		Foo: "foo",
		Bar: "bar",
	}
	sentMsgBody, err := json.Marshal(sentMsg)
	require.NoError(t, err)

	msg, err := patronSQS.NewMessageBuilder().
		QueueURL(queue).
		Body(string(sentMsgBody)).
		WithDelaySeconds(1).Body(string(sentMsgBody)).Build()
	require.NoError(t, err)

	msgID, err := pub.Publish(context.Background(), *msg)
	assert.NoError(t, err)
	assert.IsType(t, "string", msgID)

	out, err := api.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:        &queue,
		WaitTimeSeconds: aws.Int64(2),
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
}
