//go:build integration
// +build integration

package v2

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	testaws "github.com/beatlabs/patron/test/aws"
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

	api, err := testaws.CreateSQSAPI(region, endpoint)
	require.NoError(t, err)
	queue, err := testaws.CreateSQSQueue(api, queueName)
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
	assert.Equal(t, 1, testutil.CollectAndCount(publishDurationMetrics, "client_sqs_publish_duration_seconds"))
}
