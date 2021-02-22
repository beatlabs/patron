package v2

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sns/snsiface"
	"github.com/beatlabs/patron/log"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	testCases := map[string]struct {
		api         snsiface.SNSAPI
		expectedErr error
	}{
		"missing API": {api: nil, expectedErr: errors.New("missing api")},
		"success":     {api: newStubSNSAPI(nil, nil), expectedErr: nil},
	}
	for name, tC := range testCases {
		t.Run(name, func(t *testing.T) {
			p, err := New(tC.api)

			if tC.expectedErr != nil {
				assert.EqualError(t, err, tC.expectedErr.Error())
			} else {
				assert.NotNil(t, p)
				assert.NotNil(t, p.api)
			}
		})
	}
}

func Test_Publisher_Publish(t *testing.T) {
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
	ctx := context.Background()

	testCases := map[string]struct {
		sns           snsiface.SNSAPI
		expectedMsgID string
		expectedErr   string
	}{
		"publish error": {
			sns:           newStubSNSAPI(nil, errors.New("publish error")),
			expectedMsgID: "",
			expectedErr:   "failed to publish message: publish error",
		},
		"no message ID returned": {
			sns:           newStubSNSAPI(&sns.PublishOutput{}, nil),
			expectedMsgID: "",
			expectedErr:   "tried to publish a message but no message ID returned",
		},
		"success": {
			sns:           newStubSNSAPI((&sns.PublishOutput{}).SetMessageId("msgID"), nil),
			expectedMsgID: "msgID",
		},
	}
	for name, tt := range testCases {
		t.Run(name, func(t *testing.T) {
			p, err := New(tt.sns)
			require.NoError(t, err)

			msgID, err := p.Publish(ctx, &sns.PublishInput{})

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

type stubSNSAPI struct {
	snsiface.SNSAPI // Implement the interface's methods without defining all of them (just override what we need)

	output *sns.PublishOutput
	err    error
}

func newStubSNSAPI(expectedOutput *sns.PublishOutput, expectedErr error) *stubSNSAPI {
	return &stubSNSAPI{output: expectedOutput, err: expectedErr}
}

func (s *stubSNSAPI) PublishWithContext(_ context.Context, _ *sns.PublishInput, _ ...request.Option) (*sns.PublishOutput, error) {
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
		log.Fatal(err)
	}

	api := sns.New(sess)

	// Create the publisher
	pub, err := New(api)
	if err != nil {
		log.Fatal(err)
	}

	input := &sns.PublishInput{
		Message:   aws.String("my message"),
		TargetArn: nil, TopicArn: aws.String("arn:aws:sns:eu-west-1:123456789012:MyTopic"),
	}

	// Publish it
	msgID, err := pub.Publish(context.Background(), input)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(msgID)
}
