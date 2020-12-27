// +build integration

package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	v2 "github.com/beatlabs/patron/client/sns/v2"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SNS_Publish_Message(t *testing.T) {
	const topic = "test_publish_message"
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
	api, err := createSNSAPI(runtime.getSNSEndpoint())
	require.NoError(t, err)
	arn, err := createSNSTopic(api, topic)
	require.NoError(t, err)
	pub := createPublisher(t, api)
	input := &sns.PublishInput{
		Message:   aws.String(topic),
		TargetArn: aws.String(arn),
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

func createPublisher(t *testing.T, api snsiface.SNSAPI) v2.Publisher {
	p, err := v2.New(api)
	require.NoError(t, err)
	return p
}
