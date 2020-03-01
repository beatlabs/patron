package sqs

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewPublisher(t *testing.T) {
	testCases := []struct {
		desc        string
		api         sqsiface.SQSAPI
		expectedErr error
	}{
		{
			desc:        "Missing API",
			api:         nil,
			expectedErr: errors.New("missing api"),
		},
		{
			desc:        "Success",
			api:         newStubSQSAPI(nil, nil),
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
				assert.Equal(t, p.component, publisherComponent)
				assert.Equal(t, p.tag, ext.SpanKindProducer)
			}
		})
	}
}

func Test_Publisher_Publish(t *testing.T) {
	ctx := context.Background()

	msg, err := NewMessageBuilder().Body("body").QueueURL("url").Build()
	require.NoError(t, err)

	testCases := []struct {
		desc          string
		sqs           sqsiface.SQSAPI
		expectedMsgID string
		expectedErr   error
	}{
		{
			desc:          "Publish error",
			sqs:           newStubSQSAPI(nil, errors.New("publish error")),
			expectedMsgID: "",
			expectedErr:   errors.New("failed to publish message: publish error"),
		},
		{
			desc:          "No message ID returned",
			sqs:           newStubSQSAPI(&sqs.SendMessageOutput{}, nil),
			expectedMsgID: "",
			expectedErr:   errors.New("tried to publish a message but no message ID returned"),
		},
		{
			desc:          "Success",
			sqs:           newStubSQSAPI((&sqs.SendMessageOutput{}).SetMessageId("msgID"), nil),
			expectedMsgID: "msgID",
			expectedErr:   nil,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			p, err := NewPublisher(tC.sqs)
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

	msg, err := NewMessageBuilder().Body("body").QueueURL("url").Build()
	require.NoError(t, err)

	assert.Equal(t, "component url", p.publishOpName(*msg))
}

func Test_sqsHeadersCarrier_Set(t *testing.T) {
	carrier := sqsHeadersCarrier{}
	carrier.Set("foo", "bar")

	assert.Equal(t, "bar", carrier["foo"])
}

type stubSQSAPI struct {
	sqsiface.SQSAPI // Implement the interface's methods without defining all of them (just override what we need)

	output *sqs.SendMessageOutput
	err    error
}

func newStubSQSAPI(expectedOutput *sqs.SendMessageOutput, expectedErr error) *stubSQSAPI {
	return &stubSQSAPI{output: expectedOutput, err: expectedErr}
}

func (s *stubSQSAPI) SendMessageWithContext(ctx context.Context, input *sqs.SendMessageInput, options ...request.Option) (*sqs.SendMessageOutput, error) {
	return s.output, s.err
}

func ExamplePublisher() {
	// Create the SQS API with the required config, credentials, etc.
	sess, err := session.NewSession(
		aws.NewConfig().
			WithEndpoint("http://localhost:4576").
			WithRegion("eu-west-1").
			WithCredentials(
				credentials.NewStaticCredentials("aws-id", "aws-secret", "aws-token"),
			),
	)
	if err != nil {
		panic(err)
	}

	api := sqs.New(sess)

	// Create the publisher
	pub, err := NewPublisher(api)
	if err != nil {
		panic(err)
	}

	// Create a message
	msg, err := NewMessageBuilder().
		Body("message body").
		QueueURL("http://localhost:4576/queue/foo-queue").
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
