//go:build integration

package amqp

import (
	"context"
	"testing"

	"github.com/beatlabs/patron/internal/test"
	"github.com/beatlabs/patron/observability/trace"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const (
	endpoint = "amqp://bitnami:bitnami@localhost:5672/" //nolint:gosec
	queue    = "rmq-test-v2-pub-queue"
)

func TestRun(t *testing.T) {
	ctx := context.Background()

	// Setup tracing
	exp := tracetest.NewInMemoryExporter()
	tracePublisher := trace.Setup("test", nil, exp)

	shutdownProvider, collectMetrics := test.SetupMetrics(ctx, t)
	defer shutdownProvider()

	require.NoError(t, createQueue(endpoint, queue))

	pub, err := New(endpoint)
	require.NoError(t, err)

	sent := "sent"

	err = pub.Publish(ctx, "", queue, false, false,
		amqp.Publishing{ContentType: "text/plain", Body: []byte(sent)})
	require.NoError(t, err)

	require.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expected := tracetest.SpanStub{
		Name: "publish",
		Attributes: []attribute.KeyValue{
			attribute.String("exchange", ""),
			attribute.String("client", "amqp"),
		},
	}

	snaps := exp.GetSpans().Snapshots()

	assert.Len(t, snaps, 1)
	assert.Equal(t, expected.Name, snaps[0].Name())
	assert.Equal(t, expected.Attributes, snaps[0].Attributes())

	// Metrics
	_ = collectMetrics(1)

	conn, err := amqp.Dial(endpoint)
	require.NoError(t, err)

	channel, err := conn.Channel()
	require.NoError(t, err)

	dlv, err := channel.Consume(queue, "123", false, false, false, false, nil)
	require.NoError(t, err)

	var got string

	for delivery := range dlv {
		got = string(delivery.Body)
		break
	}

	assert.Equal(t, sent, got)
	require.NoError(t, channel.Close())
	require.NoError(t, conn.Close())
}

func createQueue(endpoint, queue string) error {
	conn, err := amqp.Dial(endpoint)
	if err != nil {
		return err
	}

	channel, err := conn.Channel()
	if err != nil {
		return err
	}

	_, err = channel.QueueDelete(queue, false, false, false)
	if err != nil {
		return err
	}

	_, err = channel.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return err
	}

	return nil
}
