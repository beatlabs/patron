package sqs

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding/json"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFactory(t *testing.T) {
	type args struct {
		queue     sqsiface.SQSAPI
		queueName string
		oo        []OptionFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{
				queue:     &stubQueue{},
				queueName: "queue",
				oo:        []OptionFunc{MaxMessages(1)},
			},
		},
		"missing queue": {
			args: args{
				queue:     nil,
				queueName: "queue",
				oo:        []OptionFunc{MaxMessages(1)},
			},
			expectedErr: "queue is nil",
		},
		"missing queue name": {
			args: args{
				queue:     &stubQueue{},
				queueName: "",
				oo:        []OptionFunc{MaxMessages(1)},
			},
			expectedErr: "queue name is empty",
		},
		"failed to get queue URL": {
			args: args{
				queue:     &stubQueue{getQueueURLErr: errors.New("getQueueURLErr")},
				queueName: "queue",
				oo:        []OptionFunc{MaxMessages(1)},
			},
			expectedErr: "getQueueURLErr",
		},
		"invalid option": {
			args: args{
				queue:     &stubQueue{},
				queueName: "queue",
				oo:        []OptionFunc{MaxMessages(-1)},
			},
			expectedErr: "max messages should be between 1 and 10",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewFactory(tt.args.queue, tt.args.queueName, tt.args.oo...)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestFactory_Create(t *testing.T) {
	f, err := NewFactory(&stubQueue{}, "queueName")
	require.NoError(t, err)
	got, err := f.Create()
	assert.NoError(t, err)
	cons, ok := got.(*consumer)
	assert.True(t, ok)
	assert.NotNil(t, cons.queue)
	assert.Equal(t, "queueName", cons.queueName)
	assert.Equal(t, "URL", cons.queueURL)
	assert.Equal(t, aws.Int64(3), cons.maxMessages)
	assert.Nil(t, cons.pollWaitSeconds)
	assert.Nil(t, cons.visibilityTimeout)
	assert.Equal(t, 10*time.Second, cons.statsInterval)
	assert.Nil(t, cons.cnl)
	assert.True(t, cons.OutOfOrder())
}

func Test_consumer_Consume(t *testing.T) {
	f, err := NewFactory(&stubQueue{}, "queueName", QueueStatsInterval(10*time.Millisecond))
	require.NoError(t, err)
	cns, err := f.Create()
	require.NoError(t, err)
	chMsg, chErr, err := cns.Consume(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, chMsg)
	assert.NotNil(t, chErr)
	msg := <-chMsg
	assert.NotNil(t, msg)
	time.Sleep(20 * time.Millisecond)
	require.NoError(t, cns.Close())
}

func Test_message(t *testing.T) {
	type fields struct {
		queue sqsiface.SQSAPI
	}
	tests := map[string]struct {
		fields fields
	}{
		"success, with delete": {
			fields: fields{queue: &stubQueue{}},
		},
		"success, with failed delete": {
			fields: fields{queue: &stubQueue{deleteMessageWithContextErr: errors.New("ERROR")}},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			sqsMsg := &sqs.Message{Body: aws.String(`{"key":"value"}`)}
			m := &message{
				queue:     tt.fields.queue,
				queueURL:  "queueURL",
				queueName: "queueName",
				ctx:       context.Background(),
				msg:       sqsMsg,
				span:      opentracing.StartSpan("test"),
				dec:       json.DecodeRaw,
			}
			assert.NoError(t, m.Ack())
			assert.NoError(t, m.Nack())
			assert.Equal(t, context.Background(), m.Context())
			var mp map[string]string
			assert.NoError(t, m.Decode(&mp))
			assert.Equal(t, map[string]string{"key": "value"}, mp)
			assert.Equal(t, "queueName", m.Source())
			assert.Equal(t, []byte(`{"key":"value"}`), m.Payload())
			assert.Equal(t, sqsMsg, m.Raw())
		})
	}
}

func Test_getCorrelationID(t *testing.T) {
	withID := map[string]*sqs.MessageAttributeValue{correlation.HeaderID: {StringValue: aws.String("123")}}
	withoutID := map[string]*sqs.MessageAttributeValue{correlation.HeaderID: {}}
	missingHeader := map[string]*sqs.MessageAttributeValue{}
	type args struct {
		ma map[string]*sqs.MessageAttributeValue
	}
	tests := map[string]struct {
		args args
	}{
		"with id":        {args: args{ma: withID}},
		"without id":     {args: args{ma: withoutID}},
		"missing header": {args: args{ma: missingHeader}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, getCorrelationID(tt.args.ma))
		})
	}
}

type stubQueue struct {
	sqsiface.SQSAPI
	getQueueURLErr                   error
	receiveMessageWithContextErr     error
	getQueueAttributesWithContextErr error
	deleteMessageWithContextErr      error
}

func (s stubQueue) DeleteMessageWithContext(aws.Context, *sqs.DeleteMessageInput, ...request.Option) (*sqs.DeleteMessageOutput, error) {
	if s.deleteMessageWithContextErr != nil {
		return nil, s.deleteMessageWithContextErr
	}
	return &sqs.DeleteMessageOutput{}, nil
}

func (s stubQueue) GetQueueAttributesWithContext(aws.Context, *sqs.GetQueueAttributesInput, ...request.Option) (*sqs.GetQueueAttributesOutput, error) {
	if s.getQueueAttributesWithContextErr != nil {
		return nil, s.getQueueAttributesWithContextErr
	}
	return &sqs.GetQueueAttributesOutput{
		Attributes: map[string]*string{
			sqsAttributeApproximateNumberOfMessages:           aws.String("1"),
			sqsAttributeApproximateNumberOfMessagesDelayed:    aws.String("2"),
			sqsAttributeApproximateNumberOfMessagesNotVisible: aws.String("3"),
		},
	}, nil
}

//nolint
func (s stubQueue) GetQueueUrl(*sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	if s.getQueueURLErr != nil {
		return nil, s.getQueueURLErr
	}
	return &sqs.GetQueueUrlOutput{
		QueueUrl: aws.String("URL"),
	}, nil
}

func (s stubQueue) ReceiveMessageWithContext(aws.Context, *sqs.ReceiveMessageInput, ...request.Option) (*sqs.ReceiveMessageOutput, error) {
	if s.receiveMessageWithContextErr != nil {
		return nil, s.receiveMessageWithContextErr
	}
	return &sqs.ReceiveMessageOutput{
		Messages: []*sqs.Message{
			{
				Attributes: map[string]*string{
					sqsAttributeSentTimestamp: aws.String(strconv.FormatInt(time.Now().Unix(), 10)),
				},
				Body:          aws.String(`{"key":"value"}`),
				MessageId:     aws.String("123"),
				ReceiptHandle: aws.String("123-123"),
			},
		},
	}, nil
}
