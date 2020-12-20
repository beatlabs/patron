package sqs

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

const (
	queueName = "queueName"
	queueURL  = "queueURL"
)

var mockTracer = mocktracer.New()

func TestMain(m *testing.M) {
	opentracing.SetGlobalTracer(mockTracer)
	code := m.Run()
	os.Exit(code)
}

func Test_message(t *testing.T) {
	defer mockTracer.Reset()

	ctx := context.Background()
	sp, ctx := trace.ConsumerSpan(ctx, trace.ComponentOpName(consumerComponent, queueName),
		consumerComponent, "123", nil)

	sqsAPI := &stubSQSAPI{}
	sqsMsg := &sqs.Message{}

	msg := message{
		ctx:       ctx,
		queueName: queueName,
		queueURL:  queueURL,
		queue:     sqsAPI,
		msg:       sqsMsg,
		span:      sp,
	}
	assert.Equal(t, msg.Message(), sqsMsg)
	assert.Equal(t, msg.Span(), sp)
	assert.Equal(t, msg.Context(), ctx)
}

func Test_message_ACK(t *testing.T) {
	type fields struct {
		sqsAPI sqsiface.SQSAPI
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success": {fields: fields{sqsAPI: &stubSQSAPI{}}},
		"failure": {fields: fields{sqsAPI: &stubSQSAPI{deleteMessageWithContextErr: errors.New("TEST")}}, expectedErr: "TEST"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			m := createMessage(tt.fields.sqsAPI)
			err := m.ACK()

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				expected := map[string]interface{}{
					"component":     "sqs-consumer",
					"error":         true,
					"span.kind":     ext.SpanKindEnum("consumer"),
					"version":       "dev",
					"correlationID": "123",
				}
				assert.Equal(t, expected, mockTracer.FinishedSpans()[0].Tags())
				mockTracer.Reset()
			} else {
				assert.NoError(t, err)
				expected := map[string]interface{}{
					"component":     "sqs-consumer",
					"error":         false,
					"span.kind":     ext.SpanKindEnum("consumer"),
					"version":       "dev",
					"correlationID": "123",
				}
				assert.Equal(t, expected, mockTracer.FinishedSpans()[0].Tags())
				mockTracer.Reset()
			}
		})
	}
}

func Test_message_NACK(t *testing.T) {
	defer mockTracer.Reset()

	m := createMessage(&stubSQSAPI{})

	m.NACK()
	expected := map[string]interface{}{
		"component":     "sqs-consumer",
		"error":         false,
		"span.kind":     ext.SpanKindEnum("consumer"),
		"version":       "dev",
		"correlationID": "123",
	}
	assert.Equal(t, expected, mockTracer.FinishedSpans()[0].Tags())
}

func Test_batch(t *testing.T) {
	defer mockTracer.Reset()

	sqsAPI := &stubSQSAPI{}

	msg1 := createMessage(sqsAPI)
	msg2 := createMessage(sqsAPI)

	messages := []Message{msg1, msg2}

	btc := batch{
		ctx:       context.Background(),
		queueName: queueName,
		queueURL:  queueURL,
		sqsAPI:    sqsAPI,
		messages:  []Message{msg1, msg2},
	}

	assert.EqualValues(t, btc.Messages(), messages)
}

func Test_batch_NACK(t *testing.T) {
	defer mockTracer.Reset()

	sqsAPI := &stubSQSAPI{}

	msg1 := createMessage(sqsAPI)
	msg2 := createMessage(sqsAPI)

	messages := []Message{msg1, msg2}

	btc := batch{
		ctx:       context.Background(),
		queueName: queueName,
		queueURL:  queueURL,
		sqsAPI:    sqsAPI,
		messages:  messages,
	}

	btc.NACK()

	assert.Len(t, mockTracer.FinishedSpans(), 2)
	expected := map[string]interface{}{
		"component":     "sqs-consumer",
		"error":         false,
		"span.kind":     ext.SpanKindEnum("consumer"),
		"version":       "dev",
		"correlationID": "123",
	}
	assert.Equal(t, expected, mockTracer.FinishedSpans()[0].Tags())
	assert.Equal(t, expected, mockTracer.FinishedSpans()[1].Tags())
}

func createMessage(sqsAPI sqsiface.SQSAPI) message {
	sp, ctx := trace.ConsumerSpan(context.Background(), trace.ComponentOpName(consumerComponent, queueName),
		consumerComponent, "123", nil)

	msg := message{
		ctx:       ctx,
		queueName: queueName,
		queueURL:  queueURL,
		queue:     sqsAPI,
		msg:       &sqs.Message{},
		span:      sp,
	}
	return msg
}

type stubSQSAPI struct {
	deleteMessageWithContextErr error
}

func (s stubSQSAPI) AddPermission(*sqs.AddPermissionInput) (*sqs.AddPermissionOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) AddPermissionWithContext(aws.Context, *sqs.AddPermissionInput, ...request.Option) (*sqs.AddPermissionOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) AddPermissionRequest(*sqs.AddPermissionInput) (*request.Request, *sqs.AddPermissionOutput) {
	panic("implement me")
}

func (s stubSQSAPI) ChangeMessageVisibility(*sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) ChangeMessageVisibilityWithContext(aws.Context, *sqs.ChangeMessageVisibilityInput, ...request.Option) (*sqs.ChangeMessageVisibilityOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) ChangeMessageVisibilityRequest(*sqs.ChangeMessageVisibilityInput) (*request.Request, *sqs.ChangeMessageVisibilityOutput) {
	panic("implement me")
}

func (s stubSQSAPI) ChangeMessageVisibilityBatch(*sqs.ChangeMessageVisibilityBatchInput) (*sqs.ChangeMessageVisibilityBatchOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) ChangeMessageVisibilityBatchWithContext(aws.Context, *sqs.ChangeMessageVisibilityBatchInput, ...request.Option) (*sqs.ChangeMessageVisibilityBatchOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) ChangeMessageVisibilityBatchRequest(*sqs.ChangeMessageVisibilityBatchInput) (*request.Request, *sqs.ChangeMessageVisibilityBatchOutput) {
	panic("implement me")
}

func (s stubSQSAPI) CreateQueue(*sqs.CreateQueueInput) (*sqs.CreateQueueOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) CreateQueueWithContext(aws.Context, *sqs.CreateQueueInput, ...request.Option) (*sqs.CreateQueueOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) CreateQueueRequest(*sqs.CreateQueueInput) (*request.Request, *sqs.CreateQueueOutput) {
	panic("implement me")
}

func (s stubSQSAPI) DeleteMessage(*sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) DeleteMessageWithContext(aws.Context, *sqs.DeleteMessageInput, ...request.Option) (*sqs.DeleteMessageOutput, error) {
	if s.deleteMessageWithContextErr != nil {
		return nil, s.deleteMessageWithContextErr
	}
	return &sqs.DeleteMessageOutput{}, nil
}

func (s stubSQSAPI) DeleteMessageRequest(*sqs.DeleteMessageInput) (*request.Request, *sqs.DeleteMessageOutput) {
	panic("implement me")
}

func (s stubSQSAPI) DeleteMessageBatch(*sqs.DeleteMessageBatchInput) (*sqs.DeleteMessageBatchOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) DeleteMessageBatchWithContext(aws.Context, *sqs.DeleteMessageBatchInput, ...request.Option) (*sqs.DeleteMessageBatchOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) DeleteMessageBatchRequest(*sqs.DeleteMessageBatchInput) (*request.Request, *sqs.DeleteMessageBatchOutput) {
	panic("implement me")
}

func (s stubSQSAPI) DeleteQueue(*sqs.DeleteQueueInput) (*sqs.DeleteQueueOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) DeleteQueueWithContext(aws.Context, *sqs.DeleteQueueInput, ...request.Option) (*sqs.DeleteQueueOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) DeleteQueueRequest(*sqs.DeleteQueueInput) (*request.Request, *sqs.DeleteQueueOutput) {
	panic("implement me")
}

func (s stubSQSAPI) GetQueueAttributes(*sqs.GetQueueAttributesInput) (*sqs.GetQueueAttributesOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) GetQueueAttributesWithContext(aws.Context, *sqs.GetQueueAttributesInput, ...request.Option) (*sqs.GetQueueAttributesOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) GetQueueAttributesRequest(*sqs.GetQueueAttributesInput) (*request.Request, *sqs.GetQueueAttributesOutput) {
	panic("implement me")
}

// nolint
func (s stubSQSAPI) GetQueueUrl(*sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	panic("implement me")
}

// nolint
func (s stubSQSAPI) GetQueueUrlWithContext(aws.Context, *sqs.GetQueueUrlInput, ...request.Option) (*sqs.GetQueueUrlOutput, error) {
	panic("implement me")
}

// nolint
func (s stubSQSAPI) GetQueueUrlRequest(*sqs.GetQueueUrlInput) (*request.Request, *sqs.GetQueueUrlOutput) {
	panic("implement me")
}

func (s stubSQSAPI) ListDeadLetterSourceQueues(*sqs.ListDeadLetterSourceQueuesInput) (*sqs.ListDeadLetterSourceQueuesOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) ListDeadLetterSourceQueuesWithContext(aws.Context, *sqs.ListDeadLetterSourceQueuesInput, ...request.Option) (*sqs.ListDeadLetterSourceQueuesOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) ListDeadLetterSourceQueuesRequest(*sqs.ListDeadLetterSourceQueuesInput) (*request.Request, *sqs.ListDeadLetterSourceQueuesOutput) {
	panic("implement me")
}

func (s stubSQSAPI) ListQueueTags(*sqs.ListQueueTagsInput) (*sqs.ListQueueTagsOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) ListQueueTagsWithContext(aws.Context, *sqs.ListQueueTagsInput, ...request.Option) (*sqs.ListQueueTagsOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) ListQueueTagsRequest(*sqs.ListQueueTagsInput) (*request.Request, *sqs.ListQueueTagsOutput) {
	panic("implement me")
}

func (s stubSQSAPI) ListQueues(*sqs.ListQueuesInput) (*sqs.ListQueuesOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) ListQueuesWithContext(aws.Context, *sqs.ListQueuesInput, ...request.Option) (*sqs.ListQueuesOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) ListQueuesRequest(*sqs.ListQueuesInput) (*request.Request, *sqs.ListQueuesOutput) {
	panic("implement me")
}

func (s stubSQSAPI) PurgeQueue(*sqs.PurgeQueueInput) (*sqs.PurgeQueueOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) PurgeQueueWithContext(aws.Context, *sqs.PurgeQueueInput, ...request.Option) (*sqs.PurgeQueueOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) PurgeQueueRequest(*sqs.PurgeQueueInput) (*request.Request, *sqs.PurgeQueueOutput) {
	panic("implement me")
}

func (s stubSQSAPI) ReceiveMessage(*sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) ReceiveMessageWithContext(aws.Context, *sqs.ReceiveMessageInput, ...request.Option) (*sqs.ReceiveMessageOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) ReceiveMessageRequest(*sqs.ReceiveMessageInput) (*request.Request, *sqs.ReceiveMessageOutput) {
	panic("implement me")
}

func (s stubSQSAPI) RemovePermission(*sqs.RemovePermissionInput) (*sqs.RemovePermissionOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) RemovePermissionWithContext(aws.Context, *sqs.RemovePermissionInput, ...request.Option) (*sqs.RemovePermissionOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) RemovePermissionRequest(*sqs.RemovePermissionInput) (*request.Request, *sqs.RemovePermissionOutput) {
	panic("implement me")
}

func (s stubSQSAPI) SendMessage(*sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) SendMessageWithContext(aws.Context, *sqs.SendMessageInput, ...request.Option) (*sqs.SendMessageOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) SendMessageRequest(*sqs.SendMessageInput) (*request.Request, *sqs.SendMessageOutput) {
	panic("implement me")
}

func (s stubSQSAPI) SendMessageBatch(*sqs.SendMessageBatchInput) (*sqs.SendMessageBatchOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) SendMessageBatchWithContext(aws.Context, *sqs.SendMessageBatchInput, ...request.Option) (*sqs.SendMessageBatchOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) SendMessageBatchRequest(*sqs.SendMessageBatchInput) (*request.Request, *sqs.SendMessageBatchOutput) {
	panic("implement me")
}

func (s stubSQSAPI) SetQueueAttributes(*sqs.SetQueueAttributesInput) (*sqs.SetQueueAttributesOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) SetQueueAttributesWithContext(aws.Context, *sqs.SetQueueAttributesInput, ...request.Option) (*sqs.SetQueueAttributesOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) SetQueueAttributesRequest(*sqs.SetQueueAttributesInput) (*request.Request, *sqs.SetQueueAttributesOutput) {
	panic("implement me")
}

func (s stubSQSAPI) TagQueue(*sqs.TagQueueInput) (*sqs.TagQueueOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) TagQueueWithContext(aws.Context, *sqs.TagQueueInput, ...request.Option) (*sqs.TagQueueOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) TagQueueRequest(*sqs.TagQueueInput) (*request.Request, *sqs.TagQueueOutput) {
	panic("implement me")
}

func (s stubSQSAPI) UntagQueue(*sqs.UntagQueueInput) (*sqs.UntagQueueOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) UntagQueueWithContext(aws.Context, *sqs.UntagQueueInput, ...request.Option) (*sqs.UntagQueueOutput, error) {
	panic("implement me")
}

func (s stubSQSAPI) UntagQueueRequest(*sqs.UntagQueueInput) (*request.Request, *sqs.UntagQueueOutput) {
	panic("implement me")
}
