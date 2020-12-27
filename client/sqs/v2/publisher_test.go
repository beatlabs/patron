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
	"github.com/beatlabs/patron/log"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	testCases := map[string]struct {
		api         sqsiface.SQSAPI
		expectedErr string
	}{
		"missing API": {api: nil, expectedErr: "missing api"},
		"success":     {api: newStubSQSAPI(nil, nil), expectedErr: ""},
	}
	for name, tt := range testCases {
		t.Run(name, func(t *testing.T) {
			p, err := New(tt.api)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.Equal(t, tt.api, p.api)
			}
		})
	}
}

func Test_Publisher_Publish(t *testing.T) {
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)

	ctx := context.Background()

	msg := &sqs.SendMessageInput{
		MessageBody: aws.String("body"),
		QueueUrl:    aws.String("url"),
	}

	testCases := map[string]struct {
		sqs           sqsiface.SQSAPI
		expectedMsgID string
		expectedErr   string
	}{
		"publish error": {
			sqs:           newStubSQSAPI(nil, errors.New("publish error")),
			expectedMsgID: "",
			expectedErr:   "failed to publish message: publish error",
		},
		"no message id returned": {
			sqs:           newStubSQSAPI(&sqs.SendMessageOutput{}, nil),
			expectedMsgID: "",
			expectedErr:   "tried to publish a message but no message ID returned",
		},
		"success": {
			sqs:           newStubSQSAPI((&sqs.SendMessageOutput{}).SetMessageId("msgID"), nil),
			expectedMsgID: "msgID",
			expectedErr:   "",
		},
	}
	for name, tt := range testCases {
		t.Run(name, func(t *testing.T) {
			p, err := New(tt.sqs)
			require.NoError(t, err)

			msgID, err := p.Publish(ctx, msg)

			assert.Equal(t, msgID, tt.expectedMsgID)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}
			mtr.Reset()
		})
	}
}

type stubSQSAPI struct {
	sqsiface.SQSAPI // Implement the interface's methods without defining all of them (just override what we need)

	output *sqs.SendMessageOutput
	err    error
}

func newStubSQSAPI(expectedOutput *sqs.SendMessageOutput, expectedErr error) *stubSQSAPI {
	return &stubSQSAPI{output: expectedOutput, err: expectedErr}
}

func (s *stubSQSAPI) SendMessageWithContext(_ context.Context, _ *sqs.SendMessageInput, _ ...request.Option) (*sqs.SendMessageOutput, error) {
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
		log.Fatal(err)
	}

	api := sqs.New(sess)

	pub, err := New(api)
	if err != nil {
		log.Fatal(err)
	}

	msg := &sqs.SendMessageInput{
		MessageBody: aws.String("message body"),
		QueueUrl:    aws.String("http://localhost:4576/queue/foo-queue"),
	}

	msgID, err := pub.Publish(context.Background(), msg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(msgID)
}
