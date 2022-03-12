//go:build integration
// +build integration

package v2

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/beatlabs/patron/test"
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

func Test_SNS_Publish_Message_v2(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	t.Cleanup(func() { mtr.Reset() })

	const topic = "test_publish_message_v2"

	api, err := test.CreateSNSAPI(region, endpoint)
	require.NoError(t, err)
	arn, err := test.CreateSNSTopic(api, topic)
	require.NoError(t, err)
	pub, err := New(api)
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
