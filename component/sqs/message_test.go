package sqs

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

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

var mtr = mocktracer.New()

func TestMain(m *testing.M) {
	opentracing.SetGlobalTracer(mtr)
	code := m.Run()
	os.Exit(code)
}

func Test_message(t *testing.T) {
	t.Cleanup(func() { mtr.Reset() })

	ctx := context.Background()
	sp, ctx := trace.ConsumerSpan(ctx, trace.ComponentOpName(consumerComponent, queueName),
		consumerComponent, "123", nil)

	id := "123"
	body := "body"
	sqsAPI := &stubSQSAPI{}
	sqsMsg := &sqs.Message{
		Body:      aws.String(body),
		MessageId: aws.String(id),
	}

	msg := message{
		ctx: ctx,
		queue: queue{
			name: queueName,
			url:  queueURL,
		},
		api:  sqsAPI,
		msg:  sqsMsg,
		span: sp,
	}
	assert.Equal(t, msg.Message(), sqsMsg)
	assert.Equal(t, msg.Span(), sp)
	assert.Equal(t, msg.Context(), ctx)
	assert.Equal(t, msg.ID(), id)
	assert.Equal(t, msg.Body(), []byte(body))
}

func Test_message_ACK(t *testing.T) {
	t.Cleanup(func() { mtr.Reset() })
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
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Cleanup(func() { mtr.Reset() })
			m := createMessage(tt.fields.sqsAPI, "1")
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
				assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
			} else {
				assert.NoError(t, err)
				expected := map[string]interface{}{
					"component":     "sqs-consumer",
					"error":         false,
					"span.kind":     ext.SpanKindEnum("consumer"),
					"version":       "dev",
					"correlationID": "123",
				}
				assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
			}
		})
	}
}

func Test_message_NACK(t *testing.T) {
	t.Cleanup(func() { mtr.Reset() })

	m := createMessage(&stubSQSAPI{}, "1")

	m.NACK()
	expected := map[string]interface{}{
		"component":     "sqs-consumer",
		"error":         false,
		"span.kind":     ext.SpanKindEnum("consumer"),
		"version":       "dev",
		"correlationID": "123",
	}
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
}

func Test_batch(t *testing.T) {
	t.Cleanup(func() { mtr.Reset() })

	sqsAPI := &stubSQSAPI{}

	msg1 := createMessage(sqsAPI, "1")
	msg2 := createMessage(sqsAPI, "2")

	messages := []Message{msg1, msg2}

	btc := batch{
		ctx: context.Background(),
		queue: queue{
			name: queueName,
			url:  queueURL,
		},
		sqsAPI:   sqsAPI,
		messages: []Message{msg1, msg2},
	}

	assert.EqualValues(t, btc.Messages(), messages)
}

func Test_batch_NACK(t *testing.T) {
	t.Cleanup(func() { mtr.Reset() })

	sqsAPI := &stubSQSAPI{}

	msg1 := createMessage(sqsAPI, "1")
	msg2 := createMessage(sqsAPI, "2")

	messages := []Message{msg1, msg2}

	btc := batch{
		ctx: context.Background(),
		queue: queue{
			name: queueName,
			url:  queueURL,
		},
		sqsAPI:   sqsAPI,
		messages: messages,
	}

	btc.NACK()

	assert.Len(t, mtr.FinishedSpans(), 2)
	expected := map[string]interface{}{
		"component":     "sqs-consumer",
		"error":         false,
		"span.kind":     ext.SpanKindEnum("consumer"),
		"version":       "dev",
		"correlationID": "123",
	}
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
	assert.Equal(t, expected, mtr.FinishedSpans()[1].Tags())
}

func Test_batch_ACK(t *testing.T) {
	t.Cleanup(func() { mtr.Reset() })

	msg1 := createMessage(nil, "1")
	msg2 := createMessage(nil, "2")

	messages := []Message{msg1, msg2}

	sqsAPI := &stubSQSAPI{
		succeededMessage: msg2,
		failedMessage:    msg1,
	}
	sqsAPIError := &stubSQSAPI{
		deleteMessageBatchWithContextErr: errors.New("AWS FAILURE"),
	}

	type fields struct {
		sqsAPI sqsiface.SQSAPI
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success": {
			fields: fields{sqsAPI: sqsAPI},
		},
		"AWS failure": {
			fields:      fields{sqsAPI: sqsAPIError},
			expectedErr: "AWS FAILURE",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Cleanup(func() { mtr.Reset() })
			btc := batch{
				ctx: context.Background(),
				queue: queue{
					name: queueName,
					url:  queueURL,
				},
				sqsAPI:   tt.fields.sqsAPI,
				messages: messages,
			}
			failed, err := btc.ACK()

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Len(t, mtr.FinishedSpans(), 2)
				expected := map[string]interface{}{
					"component":     "sqs-consumer",
					"error":         true,
					"span.kind":     ext.SpanKindEnum("consumer"),
					"version":       "dev",
					"correlationID": "123",
				}
				assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
				assert.Equal(t, expected, mtr.FinishedSpans()[1].Tags())
			} else {
				assert.NoError(t, err, tt)
				assert.Len(t, failed, 1)
				assert.Equal(t, msg1, failed[0])
				assert.Len(t, mtr.FinishedSpans(), 2)
				expectedSuccess := map[string]interface{}{
					"component":     "sqs-consumer",
					"error":         false,
					"span.kind":     ext.SpanKindEnum("consumer"),
					"version":       "dev",
					"correlationID": "123",
				}
				assert.Equal(t, expectedSuccess, mtr.FinishedSpans()[0].Tags())
				expectedFailure := map[string]interface{}{
					"component":     "sqs-consumer",
					"error":         true,
					"span.kind":     ext.SpanKindEnum("consumer"),
					"version":       "dev",
					"correlationID": "123",
				}
				assert.Equal(t, expectedFailure, mtr.FinishedSpans()[1].Tags())
			}
		})
	}
}

func createMessage(sqsAPI sqsiface.SQSAPI, id string) message {
	sp, ctx := trace.ConsumerSpan(context.Background(), trace.ComponentOpName(consumerComponent, queueName),
		consumerComponent, "123", nil)

	msg := message{
		ctx: ctx,
		queue: queue{
			name: queueName,
			url:  queueURL,
		},
		api: sqsAPI,
		msg: &sqs.Message{
			MessageId: aws.String(id),
		},
		span: sp,
	}
	return msg
}

type stubSQSAPI struct {
	sqsiface.SQSAPI
	receiveMessageWithContextErr     error
	deleteMessageWithContextErr      error
	deleteMessageBatchWithContextErr error
	getQueueAttributesWithContextErr error
	// nolint
	getQueueUrlWithContextErr error
	succeededMessage          Message
	failedMessage             Message
	queueURL                  string
}

func (s stubSQSAPI) DeleteMessageWithContext(aws.Context, *sqs.DeleteMessageInput, ...request.Option) (*sqs.DeleteMessageOutput, error) {
	if s.deleteMessageWithContextErr != nil {
		return nil, s.deleteMessageWithContextErr
	}
	return &sqs.DeleteMessageOutput{}, nil
}

func (s stubSQSAPI) DeleteMessageBatchWithContext(aws.Context, *sqs.DeleteMessageBatchInput, ...request.Option) (*sqs.DeleteMessageBatchOutput, error) {
	if s.deleteMessageBatchWithContextErr != nil {
		return nil, s.deleteMessageBatchWithContextErr
	}

	failed := []*sqs.BatchResultErrorEntry{{
		Code:        aws.String("1"),
		Id:          s.failedMessage.Message().MessageId,
		Message:     aws.String("ERROR"),
		SenderFault: aws.Bool(true),
	}}
	succeeded := []*sqs.DeleteMessageBatchResultEntry{{Id: s.succeededMessage.Message().MessageId}}

	return &sqs.DeleteMessageBatchOutput{
		Failed:     failed,
		Successful: succeeded,
	}, nil
}

func (s stubSQSAPI) GetQueueAttributesWithContext(aws.Context, *sqs.GetQueueAttributesInput, ...request.Option) (*sqs.GetQueueAttributesOutput, error) {
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

// nolint
func (s stubSQSAPI) GetQueueUrlWithContext(aws.Context, *sqs.GetQueueUrlInput, ...request.Option) (*sqs.GetQueueUrlOutput, error) {
	if s.getQueueUrlWithContextErr != nil {
		return nil, s.getQueueUrlWithContextErr
	}
	return &sqs.GetQueueUrlOutput{QueueUrl: aws.String(s.queueURL)}, nil
}

func (s stubSQSAPI) ReceiveMessageWithContext(aws.Context, *sqs.ReceiveMessageInput, ...request.Option) (*sqs.ReceiveMessageOutput, error) {
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
				MessageId:     s.succeededMessage.Message().MessageId,
				ReceiptHandle: aws.String("123-123"),
			},
			{
				Attributes: map[string]*string{
					sqsAttributeSentTimestamp: aws.String(strconv.FormatInt(time.Now().Unix(), 10)),
				},
				Body:          aws.String(`{"key":"value"}`),
				MessageId:     s.failedMessage.Message().MessageId,
				ReceiptHandle: aws.String("123-123"),
			},
		},
	}, nil
}
