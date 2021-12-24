package amqp

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/beatlabs/patron/trace"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

const (
	queueName = "queueName"
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

	id := "123"
	body := []byte("body")

	delivery := amqp.Delivery{MessageId: "123", Body: body}

	msg := message{
		ctx:     ctx,
		requeue: true,
		msg:     delivery,
		span:    sp,
	}
	assert.Equal(t, msg.Message(), delivery)
	assert.Equal(t, msg.Span(), sp)
	assert.Equal(t, msg.Context(), ctx)
	assert.Equal(t, msg.ID(), id)
	assert.Equal(t, msg.Body(), body)
}

func Test_message_ACK(t *testing.T) {
	t.Parallel()
	type fields struct {
		acknowledger amqp.Acknowledger
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success": {
			fields: fields{acknowledger: stubAcknowledger{}},
		},
		"failure": {
			fields:      fields{acknowledger: stubAcknowledger{ackErrors: true}},
			expectedErr: "ERROR",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := createMessage("1", tt.fields.acknowledger)
			err := m.ACK()

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				expected := map[string]interface{}{
					"component":     "amqp-consumer",
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
					"component":     "amqp-consumer",
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
	t.Parallel()
	type fields struct {
		acknowledger amqp.Acknowledger
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success": {
			fields: fields{acknowledger: stubAcknowledger{}},
		},
		"failure": {
			fields:      fields{acknowledger: stubAcknowledger{nackErrors: true}},
			expectedErr: "ERROR",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m := createMessage("1", tt.fields.acknowledger)
			err := m.NACK()

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				expected := map[string]interface{}{
					"component":     "amqp-consumer",
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
					"component":     "amqp-consumer",
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

func Test_batch_Messages(t *testing.T) {
	ackSuccess := stubAcknowledger{}
	msg1 := createMessage("1", ackSuccess)
	msg2 := createMessage("2", ackSuccess)
	messages := []Message{msg1, msg2}

	btc := batch{messages: messages}
	assert.Equal(t, messages, btc.Messages())
}

func Test_batch_ACK(t *testing.T) {
	ackSuccess := stubAcknowledger{}
	ackFailure := stubAcknowledger{ackErrors: true}

	msg1 := createMessage("1", ackSuccess)
	msg2 := createMessage("2", ackFailure)

	btc := batch{messages: []Message{msg1, msg2}}

	got, err := btc.ACK()
	assert.EqualError(t, err, "ERROR\n")
	assert.Len(t, got, 1)
	assert.Equal(t, msg2, got[0])
}

func Test_batch_NACK(t *testing.T) {
	nackSuccess := stubAcknowledger{}
	nackFailure := stubAcknowledger{nackErrors: true}

	msg1 := createMessage("1", nackSuccess)
	msg2 := createMessage("2", nackFailure)

	btc := batch{messages: []Message{msg1, msg2}}

	got, err := btc.NACK()
	assert.EqualError(t, err, "ERROR\n")
	assert.Len(t, got, 1)
	assert.Equal(t, msg2, got[0])
}

func createMessage(id string, acknowledger amqp.Acknowledger) message {
	sp, ctx := trace.ConsumerSpan(context.Background(), trace.ComponentOpName(consumerComponent, queueName),
		consumerComponent, "123", nil)

	msg := message{
		ctx: ctx,
		msg: amqp.Delivery{
			MessageId:    id,
			Acknowledger: acknowledger,
		},
		span:    sp,
		requeue: true,
	}
	return msg
}

type stubAcknowledger struct {
	ackErrors  bool
	nackErrors bool
}

func (s stubAcknowledger) Ack(_ uint64, _ bool) error {
	if s.ackErrors {
		return errors.New("ERROR")
	}
	return nil
}

func (s stubAcknowledger) Nack(_ uint64, _ bool, _ bool) error {
	if s.nackErrors {
		return errors.New("ERROR")
	}
	return nil
}

func (s stubAcknowledger) Reject(_ uint64, _ bool) error {
	panic("implement me")
}
