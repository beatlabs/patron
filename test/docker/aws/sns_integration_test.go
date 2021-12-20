//go:build integration
// +build integration

package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	v1 "github.com/beatlabs/patron/client/sns"
	v2 "github.com/beatlabs/patron/client/sns/v2"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SNS_Publish_Message(t *testing.T) {
	defer mtr.Reset()

	const topic = "test_publish_message"

	api, err := createSNSAPI(runtime.getSNSEndpoint())
	require.NoError(t, err)
	arn, err := createSNSTopic(api, topic)
	require.NoError(t, err)
	pub, err := v1.NewPublisher(api)
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

func Test_SNS_Publish_Message_v2(t *testing.T) {
	defer mtr.Reset()

	const topic = "test_publish_message_v2"

	api, err := createSNSAPI(runtime.getSNSEndpoint())
	require.NoError(t, err)
	arn, err := createSNSTopic(api, topic)
	require.NoError(t, err)
	pub, err := v2.New(api)
	require.NoError(t, err)
	input := &sns.PublishInput{
		Message:  aws.String(topic),
		TopicArn: aws.String(arn),
	}

	msgID, err := pub.Publish(context.Background(), input)
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

func createMsg(t *testing.T, topicArn string) v1.Message {
	msg, err := v1.NewMessageBuilder().Message("test msg").TopicArn(topicArn).Build()
	require.NoError(t, err)
	return *msg
}
