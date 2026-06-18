//go:build integration

package amqp

import (
	"context"
	"errors"
	"testing"

	"github.com/beatlabs/patron/correlation"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

const helperQueueName = "queueName"

func TestMessageAccessorsAndAckIntegration(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

	ctx, sp := patrontrace.StartSpan(context.Background(), patrontrace.ComponentOpName(consumerComponent, helperQueueName),
		trace.WithSpanKind(trace.SpanKindConsumer))
	msg := message{
		ctx:   ctx,
		span:  sp,
		queue: helperQueueName,
		msg: amqp.Delivery{
			MessageId:    "id",
			Body:         []byte("body"),
			Acknowledger: integrationAcknowledger{},
		},
	}

	assert.Equal(t, ctx, msg.Context())
	assert.Equal(t, "id", msg.ID())
	assert.Equal(t, []byte("body"), msg.Body())
	assert.Equal(t, sp, msg.Span())
	assert.Equal(t, amqp.Delivery{MessageId: "id", Body: []byte("body"), Acknowledger: integrationAcknowledger{}}, msg.Message())
	require.NoError(t, msg.ACK())
}

func TestMessageNackIntegration(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

	_, sp := patrontrace.StartSpan(context.Background(), patrontrace.ComponentOpName(consumerComponent, helperQueueName),
		trace.WithSpanKind(trace.SpanKindConsumer))
	msg := message{
		ctx:     context.Background(),
		span:    sp,
		queue:   helperQueueName,
		requeue: true,
		msg: amqp.Delivery{
			Acknowledger: integrationAcknowledger{},
		},
	}

	require.NoError(t, msg.NACK())
}

func TestMessageAckAndNackErrorsIntegration(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

	tests := map[string]struct {
		msgFn func(trace.Span) message
		call  func(message) error
	}{
		"ack": {
			msgFn: func(sp trace.Span) message {
				return message{ctx: context.Background(), span: sp, queue: helperQueueName, msg: amqp.Delivery{Acknowledger: integrationAcknowledger{ackErr: true}}}
			},
			call: message.ACK,
		},
		"nack": {
			msgFn: func(sp trace.Span) message {
				return message{ctx: context.Background(), span: sp, queue: helperQueueName, msg: amqp.Delivery{Acknowledger: integrationAcknowledger{nackErr: true}}}
			},
			call: message.NACK,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, sp := patrontrace.StartSpan(context.Background(), patrontrace.ComponentOpName(consumerComponent, helperQueueName),
				trace.WithSpanKind(trace.SpanKindConsumer))

			require.EqualError(t, tt.call(tt.msgFn(sp)), "ERROR")
		})
	}
}

func TestAMQPHelpersIntegration(t *testing.T) {
	t.Run("sanitize url removes credentials", func(t *testing.T) {
		assert.Equal(t, "amqp://example.com:5672/vhost", sanitizeURL("amqp://user:pass@example.com:5672/vhost"))
		assert.Equal(t, "[unparseable url]", sanitizeURL("%"))
	})

	t.Run("correlation id", func(t *testing.T) {
		assert.Equal(t, "known", getCorrelationID(amqp.Table{correlation.HeaderID: "known"}))

		got := getCorrelationID(amqp.Table{correlation.HeaderID: ""})
		_, err := uuid.Parse(got)
		require.NoError(t, err)
	})

	t.Run("carrier", func(t *testing.T) {
		delivery := amqp.Delivery{Headers: amqp.Table{"string": "value", "number": 1}}
		carrier := consumerMessageCarrier{msg: &delivery}

		assert.Equal(t, "value", carrier.Get("string"))
		assert.Empty(t, carrier.Get("missing"))
		assert.Empty(t, carrier.Get("number"))

		carrier.Set("new", "header")
		assert.Equal(t, "header", delivery.Headers["new"])
		assert.ElementsMatch(t, []string{"string", "number", "new"}, carrier.Keys())
	})

	assert.NotPanics(t, func() {
		observeQueueSize(context.Background(), helperQueueName, 3)
	})
}

type integrationAcknowledger struct {
	ackErr  bool
	nackErr bool
}

func (a integrationAcknowledger) Ack(_ uint64, _ bool) error {
	if a.ackErr {
		return errors.New("ERROR")
	}
	return nil
}

func (a integrationAcknowledger) Nack(_ uint64, _ bool, _ bool) error {
	if a.nackErr {
		return errors.New("ERROR")
	}
	return nil
}

func (a integrationAcknowledger) Reject(_ uint64, _ bool) error {
	return nil
}
