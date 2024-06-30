package sqs

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

const (
	queueName = "queueName"
	queueURL  = "queueURL"
)

func Test_message(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

	ctx := context.Background()

	ctx, sp := patrontrace.StartSpan(ctx, "123", trace.WithSpanKind(trace.SpanKindConsumer))

	id := "123"
	body := "body"
	sqsAPI := &stubSQSAPI{}
	sqsMsg := types.Message{
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
	t.Cleanup(func() { traceExporter.Reset() })

	type fields struct {
		sqsAPI API
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
			t.Cleanup(func() { traceExporter.Reset() })

			m := createMessage(tt.fields.sqsAPI, "1")
			err := m.ACK()

			require.NoError(t, tracePublisher.ForceFlush(context.Background()))

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)

				expected := createStubSpan("123", "failed to ACK message")

				got := traceExporter.GetSpans()

				assert.Len(t, got, 1)
				assertSpan(t, expected, got[0])
			} else {
				require.NoError(t, err)

				expected := createStubSpan("123", "")

				got := traceExporter.GetSpans()

				assert.Len(t, got, 1)
				assertSpan(t, expected, got[0])
			}
		})
	}
}

func Test_message_NACK(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

	m := createMessage(&stubSQSAPI{}, "1")

	m.NACK()

	require.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expected := createStubSpan("123", "")

	got := traceExporter.GetSpans()

	assert.Len(t, got, 1)
	assertSpan(t, expected, got[0])
}

func Test_batch(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

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
	t.Cleanup(func() { traceExporter.Reset() })

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

	require.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expected := createStubSpan("123", "")

	got := traceExporter.GetSpans()

	assert.Len(t, got, 2)
	assertSpan(t, expected, got[0])
	assertSpan(t, expected, got[1])
}

func Test_batch_ACK(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

	msg1 := createMessage(nil, "1")
	msg2 := createMessage(nil, "2")

	messages := []Message{msg1, msg2}

	sqsAPI := &stubSQSAPI{
		succeededMessage: msg2,
		failedMessage:    msg1,
	}
	// sqsAPIError := &stubSQSAPI{
	// 	deleteMessageBatchWithContextErr: errors.New("AWS FAILURE"),
	// }

	type fields struct {
		sqsAPI API
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success": {
			fields: fields{sqsAPI: sqsAPI},
		},
		// "AWS failure": {
		// 	fields:      fields{sqsAPI: sqsAPIError},
		// 	expectedErr: "AWS FAILURE",
		// },
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Cleanup(func() { traceExporter.Reset() })

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

			require.NoError(t, tracePublisher.ForceFlush(context.Background()))

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)

				expected := createStubSpan("123", "")

				got := traceExporter.GetSpans()

				assert.Len(t, got, 2)
				assertSpan(t, expected, got[0])
				assertSpan(t, expected, got[1])
			} else {
				require.NoError(t, err, tt)
				assert.Len(t, failed, 1)
				assert.Equal(t, msg1, failed[0])

				expectedFail := createStubSpan("123", "failed to ACK message")
				expectedSuc := createStubSpan("123", "")

				got := traceExporter.GetSpans()

				assert.Len(t, got, 2)
				assertSpan(t, expectedSuc, got[0])
				assertSpan(t, expectedFail, got[1])
			}
		})
	}
}

func createMessage(sqsAPI API, id string) message {
	ctx, sp := patrontrace.StartSpan(context.Background(), "123", trace.WithSpanKind(trace.SpanKindConsumer))

	msg := message{
		ctx: ctx,
		queue: queue{
			name: queueName,
			url:  queueURL,
		},
		api: sqsAPI,
		msg: types.Message{
			MessageId: aws.String(id),
		},
		span: sp,
	}
	return msg
}

type stubSQSAPI struct {
	API
	receiveMessageWithContextErr     error
	deleteMessageWithContextErr      error
	deleteMessageBatchWithContextErr error
	getQueueAttributesWithContextErr error
	// nolint
	getQueueUrlWithContextErr error
	succeededMessage          Message
	failedMessage             Message
	messageSent               map[string]struct{}
	queueURL                  string
}

func newStubSQSAPI() *stubSQSAPI {
	return &stubSQSAPI{
		messageSent: make(map[string]struct{}),
	}
}

func (s stubSQSAPI) DeleteMessage(_ context.Context, _ *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	if s.deleteMessageWithContextErr != nil {
		return nil, s.deleteMessageWithContextErr
	}
	return &sqs.DeleteMessageOutput{}, nil
}

func (s stubSQSAPI) DeleteMessageBatch(_ context.Context, _ *sqs.DeleteMessageBatchInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageBatchOutput, error) {
	if s.deleteMessageBatchWithContextErr != nil {
		return nil, s.deleteMessageBatchWithContextErr
	}

	failed := []types.BatchResultErrorEntry{{
		Code:        aws.String("1"),
		Id:          s.failedMessage.Message().MessageId,
		Message:     aws.String("ERROR"),
		SenderFault: true,
	}}
	succeeded := []types.DeleteMessageBatchResultEntry{{Id: s.succeededMessage.Message().MessageId}}

	return &sqs.DeleteMessageBatchOutput{
		Failed:     failed,
		Successful: succeeded,
	}, nil
}

func (s stubSQSAPI) GetQueueAttributes(_ context.Context, _ *sqs.GetQueueAttributesInput, _ ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	if s.getQueueAttributesWithContextErr != nil {
		return nil, s.getQueueAttributesWithContextErr
	}
	return &sqs.GetQueueAttributesOutput{
		Attributes: map[string]string{
			sqsAttributeApproximateNumberOfMessages:           "1",
			sqsAttributeApproximateNumberOfMessagesDelayed:    "2",
			sqsAttributeApproximateNumberOfMessagesNotVisible: "3",
		},
	}, nil
}

// nolint
func (s stubSQSAPI) GetQueueUrl(_ context.Context, _ *sqs.GetQueueUrlInput, _ ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error) {
	if s.getQueueUrlWithContextErr != nil {
		return nil, s.getQueueUrlWithContextErr
	}
	return &sqs.GetQueueUrlOutput{QueueUrl: aws.String(s.queueURL)}, nil
}

func (s stubSQSAPI) ReceiveMessage(_ context.Context, _ *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if s.receiveMessageWithContextErr != nil {
		return nil, s.receiveMessageWithContextErr
	}

	if _, ok := s.messageSent["ok"]; ok {
		return &sqs.ReceiveMessageOutput{}, nil
	}

	s.messageSent["ok"] = struct{}{}

	output := &sqs.ReceiveMessageOutput{
		Messages: []types.Message{
			{
				Attributes: map[string]string{
					sqsAttributeSentTimestamp: strconv.FormatInt(time.Now().Unix(), 10),
				},
				Body:          aws.String(`{"key":"value"}`),
				MessageId:     s.succeededMessage.Message().MessageId,
				ReceiptHandle: aws.String("123-123"),
			},
			{
				Attributes: map[string]string{
					sqsAttributeSentTimestamp: strconv.FormatInt(time.Now().Unix(), 10),
				},
				Body:          aws.String(`{"key":"value"}`),
				MessageId:     s.failedMessage.Message().MessageId,
				ReceiptHandle: aws.String("123-123"),
			},
		},
	}

	return output, nil
}

func assertSpan(t *testing.T, expected tracetest.SpanStub, got tracetest.SpanStub) {
	assert.Equal(t, expected.Name, got.Name)
	assert.Equal(t, expected.SpanKind, got.SpanKind)
	assert.Equal(t, expected.Status, got.Status)
}

func createStubSpan(name, errMsg string) tracetest.SpanStub {
	expected := tracetest.SpanStub{
		Name:     name,
		SpanKind: trace.SpanKindConsumer,
		Status: tracesdk.Status{
			Code: codes.Ok,
		},
	}

	if errMsg != "" {
		expected.Status = tracesdk.Status{
			Code:        codes.Error,
			Description: errMsg,
		}
	}

	return expected
}
