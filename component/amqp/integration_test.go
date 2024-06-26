//go:build integration

package amqp

import (
	"context"
	"testing"
	"time"

	patronamqp "github.com/beatlabs/patron/client/amqp"
	"github.com/beatlabs/patron/correlation"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

const (
	endpoint      = "amqp://bitnami:bitnami@localhost:5672/" //nolint:gosec
	rabbitMQQueue = "rmq-test-queue"
)

func TestRun(t *testing.T) {
	require.NoError(t, createQueue())

	// Setup tracing
	t.Cleanup(func() { traceExporter.Reset() })

	// Setup metrics
	read := metricsdk.NewManualReader()
	provider := metricsdk.NewMeterProvider(metricsdk.WithReader(read))
	defer func() { require.NoError(t, provider.Shutdown(context.Background())) }()
	otel.SetMeterProvider(provider)

	ctx, cnl := context.WithCancel(context.Background())

	pub, err := patronamqp.New(endpoint)
	require.NoError(t, err)

	sent := []string{"one", "two"}

	reqCtx := correlation.ContextWithID(ctx, "123")

	err = pub.Publish(reqCtx, "", rabbitMQQueue, false, false,
		amqp.Publishing{ContentType: "text/plain", Body: []byte(sent[0])})
	require.NoError(t, err)

	err = pub.Publish(reqCtx, "", rabbitMQQueue, false, false,
		amqp.Publishing{ContentType: "text/plain", Body: []byte(sent[1])})
	require.NoError(t, err)

	require.NoError(t, tracePublisher.ForceFlush(ctx))
	traceExporter.Reset()

	chReceived := make(chan []string)
	received := make([]string, 0)
	count := 0

	procFunc := func(_ context.Context, b Batch) {
		for _, msg := range b.Messages() {
			received = append(received, string(msg.Body()))
			require.NoError(t, msg.ACK())
		}

		count += len(b.Messages())
		if count == len(sent) {
			chReceived <- received
		}
	}

	cmp, err := New(endpoint, rabbitMQQueue, procFunc, WithStatsInterval(10*time.Millisecond))
	require.NoError(t, err)

	chDone := make(chan struct{})

	go func() {
		assert.NoError(t, cmp.Run(ctx))
		chDone <- struct{}{}
	}()

	got := <-chReceived
	cnl()

	<-chDone

	assert.ElementsMatch(t, sent, got)

	require.NoError(t, tracePublisher.ForceFlush(context.Background()))
	time.Sleep(time.Second)
	spans := traceExporter.GetSpans()
	assert.Len(t, spans, 2)

	expectedSpan := tracetest.SpanStub{
		Name:     "amqp rmq-test-queue",
		SpanKind: trace.SpanKindConsumer,
		Status: tracesdk.Status{
			Code: codes.Ok,
		},
	}

	assertSpan(t, expectedSpan, spans[0])
	assertSpan(t, expectedSpan, spans[1])

	// Metrics
	collectedMetrics := &metricdata.ResourceMetrics{}
	require.NoError(t, read.Collect(context.Background(), collectedMetrics))
	assert.Len(t, collectedMetrics.ScopeMetrics, 1)
	assert.Len(t, collectedMetrics.ScopeMetrics[0].Metrics, 3)
	assert.Equal(t, "amqp.publish.duration", collectedMetrics.ScopeMetrics[0].Metrics[0].Name)
	assert.Equal(t, "amqp.message.age", collectedMetrics.ScopeMetrics[0].Metrics[1].Name)
	assert.Equal(t, "amqp.message.counter", collectedMetrics.ScopeMetrics[0].Metrics[2].Name)
}

func createQueue() error {
	conn, err := amqp.Dial(endpoint)
	if err != nil {
		return err
	}

	channel, err := conn.Channel()
	if err != nil {
		return err
	}

	_, err = channel.QueueDeclare(rabbitMQQueue, true, false, false, false, nil)
	if err != nil {
		return err
	}

	return nil
}
