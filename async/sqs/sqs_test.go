package sqs

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/errors"
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
	assert.Equal(t, int64(10), cons.maxMessages)
	assert.Equal(t, int64(20), cons.pollWaitSeconds)
	assert.Equal(t, int64(30), cons.visibilityTimeout)
	assert.Equal(t, 0, cons.buffer)
	assert.Equal(t, 10*time.Second, cons.statsInterval)
	assert.Nil(t, cons.cnl)
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
			m := &message{
				queue:     tt.fields.queue,
				queueURL:  "queueURL",
				queueName: "queueName",
				ctx:       context.Background(),
				msg:       &sqs.Message{Body: aws.String(`{"key":"value"}`)},
				span:      opentracing.StartSpan("test"),
				dec:       json.DecodeRaw,
			}
			assert.NoError(t, m.Ack())
			assert.NoError(t, m.Nack())
			assert.Equal(t, context.Background(), m.Context())
			var mp map[string]string
			assert.NoError(t, m.Decode(&mp))
			assert.Equal(t, map[string]string{"key": "value"}, mp)
		})
	}
}

func Test_getCorrelationID(t *testing.T) {
	withID := map[string]*sqs.MessageAttributeValue{correlation.HeaderID: &sqs.MessageAttributeValue{StringValue: aws.String("123")}}
	withoutID := map[string]*sqs.MessageAttributeValue{correlation.HeaderID: &sqs.MessageAttributeValue{}}
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
	getQueueURLErr                   error
	receiveMessageWithContextErr     error
	getQueueAttributesWithContextErr error
	deleteMessageWithContextErr      error
}

func (s stubQueue) AddPermission(*sqs.AddPermissionInput) (*sqs.AddPermissionOutput, error) {
	panic("implement me")
}

func (s stubQueue) AddPermissionWithContext(aws.Context, *sqs.AddPermissionInput, ...request.Option) (*sqs.AddPermissionOutput, error) {
	panic("implement me")
}

func (s stubQueue) AddPermissionRequest(*sqs.AddPermissionInput) (*request.Request, *sqs.AddPermissionOutput) {
	panic("implement me")
}

func (s stubQueue) ChangeMessageVisibility(*sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error) {
	panic("implement me")
}

func (s stubQueue) ChangeMessageVisibilityWithContext(aws.Context, *sqs.ChangeMessageVisibilityInput, ...request.Option) (*sqs.ChangeMessageVisibilityOutput, error) {
	panic("implement me")
}

func (s stubQueue) ChangeMessageVisibilityRequest(*sqs.ChangeMessageVisibilityInput) (*request.Request, *sqs.ChangeMessageVisibilityOutput) {
	panic("implement me")
}

func (s stubQueue) ChangeMessageVisibilityBatch(*sqs.ChangeMessageVisibilityBatchInput) (*sqs.ChangeMessageVisibilityBatchOutput, error) {
	panic("implement me")
}

func (s stubQueue) ChangeMessageVisibilityBatchWithContext(aws.Context, *sqs.ChangeMessageVisibilityBatchInput, ...request.Option) (*sqs.ChangeMessageVisibilityBatchOutput, error) {
	panic("implement me")
}

func (s stubQueue) ChangeMessageVisibilityBatchRequest(*sqs.ChangeMessageVisibilityBatchInput) (*request.Request, *sqs.ChangeMessageVisibilityBatchOutput) {
	panic("implement me")
}

func (s stubQueue) CreateQueue(*sqs.CreateQueueInput) (*sqs.CreateQueueOutput, error) {
	panic("implement me")
}

func (s stubQueue) CreateQueueWithContext(aws.Context, *sqs.CreateQueueInput, ...request.Option) (*sqs.CreateQueueOutput, error) {
	panic("implement me")
}

func (s stubQueue) CreateQueueRequest(*sqs.CreateQueueInput) (*request.Request, *sqs.CreateQueueOutput) {
	panic("implement me")
}

func (s stubQueue) DeleteMessage(*sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	panic("implement me")
}

func (s stubQueue) DeleteMessageWithContext(aws.Context, *sqs.DeleteMessageInput, ...request.Option) (*sqs.DeleteMessageOutput, error) {
	if s.deleteMessageWithContextErr != nil {
		return nil, s.deleteMessageWithContextErr
	}
	return &sqs.DeleteMessageOutput{}, nil
}

func (s stubQueue) DeleteMessageRequest(*sqs.DeleteMessageInput) (*request.Request, *sqs.DeleteMessageOutput) {
	panic("implement me")
}

func (s stubQueue) DeleteMessageBatch(*sqs.DeleteMessageBatchInput) (*sqs.DeleteMessageBatchOutput, error) {
	panic("implement me")
}

func (s stubQueue) DeleteMessageBatchWithContext(aws.Context, *sqs.DeleteMessageBatchInput, ...request.Option) (*sqs.DeleteMessageBatchOutput, error) {
	panic("implement me")
}

func (s stubQueue) DeleteMessageBatchRequest(*sqs.DeleteMessageBatchInput) (*request.Request, *sqs.DeleteMessageBatchOutput) {
	panic("implement me")
}

func (s stubQueue) DeleteQueue(*sqs.DeleteQueueInput) (*sqs.DeleteQueueOutput, error) {
	panic("implement me")
}

func (s stubQueue) DeleteQueueWithContext(aws.Context, *sqs.DeleteQueueInput, ...request.Option) (*sqs.DeleteQueueOutput, error) {
	panic("implement me")
}

func (s stubQueue) DeleteQueueRequest(*sqs.DeleteQueueInput) (*request.Request, *sqs.DeleteQueueOutput) {
	panic("implement me")
}

func (s stubQueue) GetQueueAttributes(*sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	panic("implement me")
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

func (s stubQueue) GetQueueAttributesRequest(*sqs.GetQueueAttributesInput) (*request.Request, *sqs.GetQueueAttributesOutput) {
	panic("implement me")
}

func (s stubQueue) GetQueueUrl(*sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	if s.getQueueURLErr != nil {
		return nil, s.getQueueURLErr
	}
	return &sqs.GetQueueUrlOutput{
		QueueUrl: aws.String("URL"),
	}, nil
}

func (s stubQueue) GetQueueUrlWithContext(aws.Context, *sqs.GetQueueUrlInput, ...request.Option) (*sqs.GetQueueUrlOutput, error) {
	panic("implement me")
}

func (s stubQueue) GetQueueUrlRequest(*sqs.GetQueueUrlInput) (*request.Request, *sqs.GetQueueUrlOutput) {
	panic("implement me")
}

func (s stubQueue) ListDeadLetterSourceQueues(*sqs.ListDeadLetterSourceQueuesInput) (*sqs.ListDeadLetterSourceQueuesOutput, error) {
	panic("implement me")
}

func (s stubQueue) ListDeadLetterSourceQueuesWithContext(aws.Context, *sqs.ListDeadLetterSourceQueuesInput, ...request.Option) (*sqs.ListDeadLetterSourceQueuesOutput, error) {
	panic("implement me")
}

func (s stubQueue) ListDeadLetterSourceQueuesRequest(*sqs.ListDeadLetterSourceQueuesInput) (*request.Request, *sqs.ListDeadLetterSourceQueuesOutput) {
	panic("implement me")
}

func (s stubQueue) ListQueueTags(*sqs.ListQueueTagsInput) (*sqs.ListQueueTagsOutput, error) {
	panic("implement me")
}

func (s stubQueue) ListQueueTagsWithContext(aws.Context, *sqs.ListQueueTagsInput, ...request.Option) (*sqs.ListQueueTagsOutput, error) {
	panic("implement me")
}

func (s stubQueue) ListQueueTagsRequest(*sqs.ListQueueTagsInput) (*request.Request, *sqs.ListQueueTagsOutput) {
	panic("implement me")
}

func (s stubQueue) ListQueues(*sqs.ListQueuesInput) (*sqs.ListQueuesOutput, error) {
	panic("implement me")
}

func (s stubQueue) ListQueuesWithContext(aws.Context, *sqs.ListQueuesInput, ...request.Option) (*sqs.ListQueuesOutput, error) {
	panic("implement me")
}

func (s stubQueue) ListQueuesRequest(*sqs.ListQueuesInput) (*request.Request, *sqs.ListQueuesOutput) {
	panic("implement me")
}

func (s stubQueue) PurgeQueue(*sqs.PurgeQueueInput) (*sqs.PurgeQueueOutput, error) {
	panic("implement me")
}

func (s stubQueue) PurgeQueueWithContext(aws.Context, *sqs.PurgeQueueInput, ...request.Option) (*sqs.PurgeQueueOutput, error) {
	panic("implement me")
}

func (s stubQueue) PurgeQueueRequest(*sqs.PurgeQueueInput) (*request.Request, *sqs.PurgeQueueOutput) {
	panic("implement me")
}

func (s stubQueue) ReceiveMessage(*sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	panic("implement me")
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

func (s stubQueue) ReceiveMessageRequest(*sqs.ReceiveMessageInput) (*request.Request, *sqs.ReceiveMessageOutput) {
	panic("implement me")
}

func (s stubQueue) RemovePermission(*sqs.RemovePermissionInput) (*sqs.RemovePermissionOutput, error) {
	panic("implement me")
}

func (s stubQueue) RemovePermissionWithContext(aws.Context, *sqs.RemovePermissionInput, ...request.Option) (*sqs.RemovePermissionOutput, error) {
	panic("implement me")
}

func (s stubQueue) RemovePermissionRequest(*sqs.RemovePermissionInput) (*request.Request, *sqs.RemovePermissionOutput) {
	panic("implement me")
}

func (s stubQueue) SendMessage(*sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	panic("implement me")
}

func (s stubQueue) SendMessageWithContext(aws.Context, *sqs.SendMessageInput, ...request.Option) (*sqs.SendMessageOutput, error) {
	panic("implement me")
}

func (s stubQueue) SendMessageRequest(*sqs.SendMessageInput) (*request.Request, *sqs.SendMessageOutput) {
	panic("implement me")
}

func (s stubQueue) SendMessageBatch(*sqs.SendMessageBatchInput) (*sqs.SendMessageBatchOutput, error) {
	panic("implement me")
}

func (s stubQueue) SendMessageBatchWithContext(aws.Context, *sqs.SendMessageBatchInput, ...request.Option) (*sqs.SendMessageBatchOutput, error) {
	panic("implement me")
}

func (s stubQueue) SendMessageBatchRequest(*sqs.SendMessageBatchInput) (*request.Request, *sqs.SendMessageBatchOutput) {
	panic("implement me")
}

func (s stubQueue) SetQueueAttributes(*sqs.SetQueueAttributesInput) (*sqs.SetQueueAttributesOutput, error) {
	panic("implement me")
}

func (s stubQueue) SetQueueAttributesWithContext(aws.Context, *sqs.SetQueueAttributesInput, ...request.Option) (*sqs.SetQueueAttributesOutput, error) {
	panic("implement me")
}

func (s stubQueue) SetQueueAttributesRequest(*sqs.SetQueueAttributesInput) (*request.Request, *sqs.SetQueueAttributesOutput) {
	panic("implement me")
}

func (s stubQueue) TagQueue(*sqs.TagQueueInput) (*sqs.TagQueueOutput, error) {
	panic("implement me")
}

func (s stubQueue) TagQueueWithContext(aws.Context, *sqs.TagQueueInput, ...request.Option) (*sqs.TagQueueOutput, error) {
	panic("implement me")
}

func (s stubQueue) TagQueueRequest(*sqs.TagQueueInput) (*request.Request, *sqs.TagQueueOutput) {
	panic("implement me")
}

func (s stubQueue) UntagQueue(*sqs.UntagQueueInput) (*sqs.UntagQueueOutput, error) {
	panic("implement me")
}

func (s stubQueue) UntagQueueWithContext(aws.Context, *sqs.UntagQueueInput, ...request.Option) (*sqs.UntagQueueOutput, error) {
	panic("implement me")
}

func (s stubQueue) UntagQueueRequest(*sqs.UntagQueueInput) (*request.Request, *sqs.UntagQueueOutput) {
	panic("implement me")
}
