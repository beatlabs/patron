// +build integration

package sns

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// These values are taken from examples/docker-compose.yml
	testSnsEndpoint string = "http://localhost:4575"
	testSnsRegion   string = "eu-west-1"
)

func Test_Publish_Message(t *testing.T) {
	api := createAPI(t)
	topicArn := createTopic(t, api)
	pub := createPublisher(t, api)
	msg := createMsg(t, topicArn)

	msgID, err := pub.Publish(context.Background(), msg)
	assert.NoError(t, err)
	assert.IsType(t, "string", msgID)
}

func createAPI(t *testing.T) snsiface.SNSAPI {
	sess, err := session.NewSession(
		aws.NewConfig().
			WithEndpoint(testSnsEndpoint).
			WithRegion(testSnsRegion).
			WithCredentials(credentials.NewStaticCredentials("test", "test", "")),
	)
	require.NoError(t, err)

	cfg := &aws.Config{
		Region: aws.String(testSnsRegion),
	}

	return sns.New(sess, cfg)
}

func createTopic(t *testing.T, api snsiface.SNSAPI) (topicArn string) {
	out, err := api.CreateTopic(&sns.CreateTopicInput{
		Name: aws.String("test-topic"),
	})
	require.NoError(t, err)

	return *out.TopicArn
}

func createPublisher(t *testing.T, api snsiface.SNSAPI) Publisher {
	p, err := NewPublisher(api)
	require.NoError(t, err)

	return p
}

func createMsg(t *testing.T, topicArn string) Message {
	b := NewMessageBuilder()

	msg, err := b.
		Message("test msg").
		TopicArn(topicArn).
		Build()
	require.NoError(t, err)

	return *msg
}
