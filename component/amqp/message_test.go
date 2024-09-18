package amqp

import (
	"context"
	"errors"
	"os"
	"testing"

	patrontrace "github.com/beatlabs/patron/observability/trace"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

const (
	queueName = "queueName"
)

var (
	tracePublisher *tracesdk.TracerProvider
	traceExporter  = tracetest.NewInMemoryExporter()
)

func TestMain(m *testing.M) {
	os.Setenv("OTEL_BSP_SCHEDULE_DELAY", "100")

	tracePublisher = patrontrace.Setup("test", nil, traceExporter)

	os.Exit(m.Run())
}

func Test_message(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })
	ctx, sp := patrontrace.StartSpan(context.Background(), "test")

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
	t.Cleanup(func() { traceExporter.Reset() })

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
		t.Run(name, func(t *testing.T) {
			t.Cleanup(func() { traceExporter.Reset() })
			m := createMessage("1", tt.fields.acknowledger)
			err := m.ACK()

			require.NoError(t, tracePublisher.ForceFlush(context.Background()))

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)

				expected := tracetest.SpanStub{
					Name:     "amqp queueName",
					SpanKind: trace.SpanKindConsumer,
					Status: tracesdk.Status{
						Code:        codes.Error,
						Description: "failed to ACK message",
					},
				}

				got := traceExporter.GetSpans()

				assert.Len(t, got, 1)
				assertSpan(t, expected, got[0])
			} else {
				require.NoError(t, err)

				expected := tracetest.SpanStub{
					Name:     "amqp queueName",
					SpanKind: trace.SpanKindConsumer,
					Status: tracesdk.Status{
						Code: codes.Ok,
					},
				}

				got := traceExporter.GetSpans()

				assert.Len(t, got, 1)
				assertSpan(t, expected, got[0])
			}
		})
	}
}

func Test_message_NACK(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

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
		t.Run(name, func(t *testing.T) {
			t.Cleanup(func() { traceExporter.Reset() })
			m := createMessage("1", tt.fields.acknowledger)
			err := m.NACK()

			require.NoError(t, tracePublisher.ForceFlush(context.Background()))

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)

				expected := tracetest.SpanStub{
					Name:     "amqp queueName",
					SpanKind: trace.SpanKindConsumer,
					Status: tracesdk.Status{
						Code:        codes.Error,
						Description: "failed to NACK message",
					},
				}

				got := traceExporter.GetSpans()

				assert.Len(t, got, 1)
				assertSpan(t, expected, got[0])
			} else {
				require.NoError(t, err)
				expected := tracetest.SpanStub{
					Name:     "amqp queueName",
					SpanKind: trace.SpanKindConsumer,
					Status: tracesdk.Status{
						Code: codes.Ok,
					},
				}

				got := traceExporter.GetSpans()

				assert.Len(t, got, 1)
				assertSpan(t, expected, got[0])
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
	require.EqualError(t, err, "ERROR")
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
	require.EqualError(t, err, "ERROR")
	assert.Len(t, got, 1)
	assert.Equal(t, msg2, got[0])
}

func createMessage(id string, acknowledger amqp.Acknowledger) message {
	ctx, sp := patrontrace.StartSpan(context.Background(),
		patrontrace.ComponentOpName(consumerComponent, queueName), trace.WithSpanKind(trace.SpanKindConsumer))

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

func assertSpan(t *testing.T, expected tracetest.SpanStub, got tracetest.SpanStub) {
	assert.Equal(t, expected.Name, got.Name)
	assert.Equal(t, expected.SpanKind, got.SpanKind)
	assert.Equal(t, expected.Status, got.Status)
}
