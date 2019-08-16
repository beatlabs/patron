// Package sns provides a set of common interfaces and structs for publishing messages to AWS SNS. Implementations
// in this package also include tracing capabilities by default.

package sns

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
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

	assert.Equal(t, "component publish:unknown", p.publishOpName(*msg))
}

type stubSNSAPI struct {
	snsiface.SNSAPI // Implement the interface's methods without defining all of them (just override what we need)

	output *sns.PublishOutput
	err    error
}

func newStubSNSAPI(expectedOutput *sns.PublishOutput, expectedErr error) *stubSNSAPI {
	return &stubSNSAPI{output: expectedOutput, err: expectedErr}
}

func (s *stubSNSAPI) Publish(input *sns.PublishInput) (*sns.PublishOutput, error) {
	return s.output, s.err
}

func ExamplePublisher() {
	// Create the SNS API with the required config, credentials, etc.
	var sess = session.Must(session.NewSession(
		aws.NewConfig().WithEndpoint("http://localhost:4575"),
	))
	cfg := &aws.Config{
		Region: aws.String("eu-west-1"),
	}
	api := sns.New(sess, cfg)

	// Create the publisher
	pub, err := NewPublisher(api)
	if err != nil {
		panic(err)
	}

	// Create a message
	msg, err := NewMessageBuilder().
		WithMessage("my message").
		WithTopicARN("arn:aws:sns:eu-west-1:123456789012:MyTopic").
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
