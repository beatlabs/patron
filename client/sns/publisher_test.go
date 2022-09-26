package sns

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/beatlabs/patron/log"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	testCases := map[string]struct {
		api         API
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
		sns           API
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
			sns:           newStubSNSAPI((&sns.PublishOutput{MessageId: aws.String("msgID")}), nil),
			expectedMsgID: "msgID",
		},
	}
	for name, tt := range testCases {
		t.Run(name, func(t *testing.T) {
			p, err := New(tt.sns)
			require.NoError(t, err)

			msgID, err := p.Publish(ctx, &sns.PublishInput{
				TopicArn: aws.String("123"),
			})

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
	API // Implement the interface's methods without defining all of them (just override what we need)

	output *sns.PublishOutput
	err    error
}

func newStubSNSAPI(expectedOutput *sns.PublishOutput, expectedErr error) *stubSNSAPI {
	return &stubSNSAPI{output: expectedOutput, err: expectedErr}
}

func (s *stubSNSAPI) Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error) {
	return s.output, s.err
}

func ExamplePublisher() {
	// Create the SNS API with the required config, credentials, etc.
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == sns.ServiceID && region == "eu-west-1" {
			return aws.Endpoint{
				URL:           "http://localhost:4575",
				SigningRegion: "eu-west-1",
			}, nil
		}
		// returning EndpointNotFoundError will allow the service to fallback to it's default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-west-1"),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("aws-id", "aws-secret", "aws-token"))),
	)
	if err != nil {
		log.Fatal(err)
	}

	api := sns.NewFromConfig(cfg)

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
