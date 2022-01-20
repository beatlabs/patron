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
	"github.com/beatlabs/patron/correlation"
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
		sqs           *stubSQSAPI
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

func Test_Publisher_Publish_InjectsHeaders(t *testing.T) {
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)

	correlationID := "correlationID"
	ctx := correlation.ContextWithID(context.Background(), correlationID)

	msg := sqs.SendMessageInput{
		MessageBody: aws.String("body"),
		QueueUrl:    aws.String("url"),
	}

	sqsStub := newStubSQSAPI((&sqs.SendMessageOutput{}).SetMessageId("msgID"), nil)
	p, err := New(sqsStub)
	require.NoError(t, err)

	// Mimic the opentracing injector using a mocked one.
	mockTracerInjector := NewMockTracerInjector()
	mtr.RegisterInjector(opentracing.TextMap, mockTracerInjector)

	expectedMsgInput := msg
	expectedMsgInput.MessageAttributes = map[string]*sqs.MessageAttributeValue{
		// Expect the opentracing headers to be injected.
		mockTracerInjector.headerKey: {
			StringValue: aws.String(mockTracerInjector.headerValue),
			DataType:    aws.String("String"),
		},

		// Expect the correlation header to be injected.
		correlation.HeaderID: {
			StringValue: aws.String(correlationID),
			DataType:    aws.String("String"),
		},
	}

	t.Run("sets correlation ID and opentracing headers", func(t *testing.T) {
		sqsStub.expectMessageInput(t, &expectedMsgInput)

		_, err = p.Publish(ctx, &msg)
		require.NoError(t, err)

		mtr.Reset()
	})

	t.Run("does not set correlation ID header when it's already present", func(t *testing.T) {
		msg.MessageAttributes = map[string]*sqs.MessageAttributeValue{
			correlation.HeaderID: {
				StringValue: aws.String("something"),
				DataType:    aws.String("String"),
			},
		}

		// Expect the original value to be retained.
		expectedMsgInput.MessageAttributes[correlation.HeaderID] = msg.MessageAttributes[correlation.HeaderID]

		sqsStub.expectMessageInput(t, &expectedMsgInput)

		_, err = p.Publish(ctx, &msg)
		require.NoError(t, err)

		mtr.Reset()
	})
}

type stubSQSAPI struct {
	sqsiface.SQSAPI // Implement the interface's methods without defining all of them (just override what we need)

	output *sqs.SendMessageOutput
	err    error

	expectedMsgInput *sqs.SendMessageInput
	t                *testing.T
}

func newStubSQSAPI(expectedOutput *sqs.SendMessageOutput, expectedErr error) *stubSQSAPI {
	return &stubSQSAPI{output: expectedOutput, err: expectedErr}
}

func (s *stubSQSAPI) SendMessageWithContext(
	_ context.Context, actualMessage *sqs.SendMessageInput, _ ...request.Option,
) (*sqs.SendMessageOutput, error) {
	if s.expectedMsgInput != nil {
		assert.Equal(s.t, s.expectedMsgInput, actualMessage)
	}

	return s.output, s.err
}

func (s *stubSQSAPI) expectMessageInput(t *testing.T, expectedMsgInput *sqs.SendMessageInput) {
	s.t = t
	s.expectedMsgInput = expectedMsgInput
}

type MockTracerInjector struct {
	mocktracer.Injector

	headerKey   string
	headerValue string
}

func (i MockTracerInjector) Inject(_ mocktracer.MockSpanContext, carrier interface{}) error {
	writer, ok := carrier.(opentracing.TextMapWriter)
	if !ok {
		return fmt.Errorf("unexpected carrier")
	}
	writer.Set(i.headerKey, i.headerValue)
	return nil
}

func NewMockTracerInjector() MockTracerInjector {
	return MockTracerInjector{
		headerKey:   "header-injected-by",
		headerValue: "mock-injector",
	}
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
