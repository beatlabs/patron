package sns

import (
	"context"
	"testing"

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
	cfg, err := NewConfig(session.New())
	require.NoError(t, err)

	p := NewPublisher(*cfg)
	assert.IsType(t, &sns.SNS{}, p.sns)
	assert.Equal(t, p.component, trace.SNSPublisherComponent)
	assert.Equal(t, p.tag, ext.SpanKindProducer)
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
			p := TracedPublisher{sns: tC.sns}
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
