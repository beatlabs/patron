//go:build integration
// +build integration

package sns

import (
	"context"
	"testing"

	testaws "github.com/beatlabs/patron/test/aws"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	region   = "eu-west-1"
	endpoint = "http://localhost:4566"
)

func Test_SNS_Publish_Message(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	t.Cleanup(func() { mtr.Reset() })

	const topic = "test_publish_message"
	api, err := testaws.CreateSNSAPI(region, endpoint)
	require.NoError(t, err)
	arn, err := testaws.CreateSNSTopic(api, topic)
	require.NoError(t, err)
	pub, err := NewPublisher(api)
	require.NoError(t, err)
	msg := createMsg(t, arn)

	msgID, err := pub.Publish(context.Background(), msg)
	assert.NoError(t, err)
	assert.IsType(t, "string", msgID)
	expected := map[string]interface{}{
		"component": "sns-publisher",
		"error":     false,
		"span.kind": ext.SpanKindEnum("producer"),
		"version":   "dev",
	}
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
}

func createMsg(t *testing.T, topicArn string) Message {
	msg, err := NewMessageBuilder().Message("test msg").TopicArn(topicArn).Build()
	require.NoError(t, err)
	return *msg
}
