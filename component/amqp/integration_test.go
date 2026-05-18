//go:build integration

package amqp

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	patronamqp "github.com/beatlabs/patron/client/amqp"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/internal/test"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/goleak"
)

var (
	tracePublisher *tracesdk.TracerProvider
	traceExporter  = tracetest.NewInMemoryExporter()
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"),
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
	)
}

func init() {
	if err := os.Setenv("OTEL_BSP_SCHEDULE_DELAY", "100"); err != nil {
		panic(err)
	}

	tracePublisher = patrontrace.Setup("test", nil, traceExporter)
}

const (
	endpoint      = "amqp://bitnami:bitnami@localhost:5672/" //nolint:gosec
	rabbitMQQueue = "rmq-test-queue"
	missingQueue  = "rmq-missing-queue"
)

func TestRun(t *testing.T) {
	require.NoError(t, createQueue())

	// Setup tracing
	t.Cleanup(func() { traceExporter.Reset() })

	ctx, cnl := context.WithCancel(context.Background())

	shutdownProvider, collectMetrics := test.SetupMetrics(context.Background(), t)
	defer shutdownProvider()

	pub, err := patronamqp.New(endpoint)
	require.NoError(t, err)
	defer func() {
		if pub != nil {
			_ = pub.Close() // Ignore close errors in cleanup
		}
	}()
	defer cnl()

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

	test.AssertSpan(t, expectedSpan, spans[0])
	test.AssertSpan(t, expectedSpan, spans[1])

	// Metrics
	collectedMetrics := collectMetrics(3)
	test.AssertMetric(t, collectedMetrics.ScopeMetrics[0].Metrics, "amqp.publish.duration")
	test.AssertMetric(t, collectedMetrics.ScopeMetrics[0].Metrics, "amqp.message.age")
	test.AssertMetric(t, collectedMetrics.ScopeMetrics[0].Metrics, "amqp.message.counter")
}

func TestProcessLoop_LogsStatsError(t *testing.T) {
	require.NoError(t, createQueue())

	cmp, err := New(endpoint, rabbitMQQueue, func(_ context.Context, _ Batch) {},
		WithStatsInterval(10*time.Millisecond),
		WithRetry(1, time.Millisecond),
	)
	require.NoError(t, err)

	sub, err := cmp.subscribe()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sub.close()
	})

	cmp.queueCfg.queue = missingQueue

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	errOutput := captureAMQPIntegrationLogOutput(t, func() {
		err = cmp.processLoop(ctx, sub)
	})

	require.Error(t, err)
	assert.Contains(t, errOutput, "failed to report amqp stats")
	assert.NotContains(t, errOutput, "%v")
}

func captureAMQPIntegrationLogOutput(t *testing.T, fn func()) string {
	t.Helper()

	originalLogger := slog.Default()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	defer func() {
		slog.SetDefault(originalLogger)
	}()

	slog.SetDefault(logger)
	fn()

	return buf.String()
}

func createQueue() error {
	conn, err := amqp.Dial(endpoint)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	channel, err := conn.Channel()
	if err != nil {
		return err
	}
	defer func() { _ = channel.Close() }()

	_, err = channel.QueueDeclare(rabbitMQQueue, true, false, false, false, nil)
	if err != nil {
		return err
	}

	return nil
}
