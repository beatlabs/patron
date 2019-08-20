package sns

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewPublisher(t *testing.T) {
	testCases := []struct {
		desc        string
		api         snsiface.SNSAPI
		expectedErr error
	}{
		{
			desc:        "Missing API",
			api:         nil,
			expectedErr: errors.New("missing api"),
		},
		{
			desc:        "Success",
			api:         newStubSNSAPI(nil, nil),
			expectedErr: nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			p, err := NewPublisher(tC.api)

			if tC.expectedErr != nil {
				assert.Nil(t, p)
				assert.EqualError(t, err, tC.expectedErr.Error())
			} else {
				assert.Equal(t, tC.api, p.api)
				assert.Equal(t, p.component, trace.SNSPublisherComponent)
				assert.Equal(t, p.tag, ext.SpanKindProducer)
			}
		})
	}
}

func Test_Publisher_Publish(t *testing.T) {
	ctx := context.Background()

	msg, err := NewMessageBuilder().Build()
	require.NoError(t, err)

	testCases := []struct {
		desc          string
		sns           snsiface.SNSAPI
		expectedMsgID string
		expectedErr   error
	}{
		{
			desc:          "Publish error",
			sns:           newStubSNSAPI(nil, errors.New("publish error")),
			expectedMsgID: "",
			expectedErr:   errors.New("failed to publish message: publish error"),
		},
		{
			desc:          "No message ID returned",
			sns:           newStubSNSAPI(&sns.PublishOutput{}, nil),
			expectedMsgID: "",
			expectedErr:   errors.New("tried to publish a message but no message ID returned"),
		},
		{
			desc:          "Success",
			sns:           newStubSNSAPI((&sns.PublishOutput{}).SetMessageId("msgID"), nil),
			expectedMsgID: "msgID",
			expectedErr:   nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			p, err := NewPublisher(tC.sns)
			require.NoError(t, err)

			msgID, err := p.Publish(ctx, *msg)

			assert.Equal(t, msgID, tC.expectedMsgID)

			if tC.expectedErr != nil {
				assert.EqualError(t, err, tC.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_Publisher_publishOpName(t *testing.T) {
	component := "component"
	p := &TracedPublisher{
		component: component,
	}

	msg, err := NewMessageBuilder().Build()
	require.NoError(t, err)

	assert.Equal(t, "component unknown", p.publishOpName(*msg))
}

func Test_snsHeadersCarrier_Set(t *testing.T) {
	carrier := snsHeadersCarrier{}
	carrier.Set("foo", "bar")

	assert.Equal(t, "bar", carrier["foo"])
}

type stubSNSAPI struct {
	snsiface.SNSAPI // Implement the interface's methods without defining all of them (just override what we need)

	output *sns.PublishOutput
	err    error
}

func newStubSNSAPI(expectedOutput *sns.PublishOutput, expectedErr error) *stubSNSAPI {
	return &stubSNSAPI{output: expectedOutput, err: expectedErr}
}

func (s *stubSNSAPI) PublishWithContext(ctx context.Context, input *sns.PublishInput, options ...request.Option) (*sns.PublishOutput, error) {
	return s.output, s.err
}

func ExamplePublisher() {
	// Create the SNS API with the required config, credentials, etc.
	sess, err := session.NewSession(
		aws.NewConfig().
			WithEndpoint("http://localhost:4575").
			WithRegion("eu-west-1").
			WithCredentials(
				credentials.NewStaticCredentials("aws-id", "aws-secret", "aws-token"),
			),
	)
	if err != nil {
		panic(err)
	}

	api := sns.New(sess)

	// Create the publisher
	pub, err := NewPublisher(api)
	if err != nil {
		panic(err)
	}

	// Create a message
	msg, err := NewMessageBuilder().
		Message("my message").
		TopicArn("arn:aws:sns:eu-west-1:123456789012:MyTopic").
		Build()
	if err != nil {
		panic(err)
	}

	// Publish it
	msgID, err := pub.Publish(context.Background(), *msg)
	if err != nil {
		panic(err)
	}

	fmt.Println(msgID)
}
