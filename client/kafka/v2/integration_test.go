//go:build integration

package v2

import (
	"context"
	"os"
	"testing"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron/internal/test"
	"github.com/beatlabs/patron/observability/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const (
	clientTopic = "clientTopic"
)

var (
	brokers        = []string{"127.0.0.1:9092"}
	tracePublisher *sdktrace.TracerProvider
	traceExporter  *tracetest.InMemoryExporter
)

func TestMain(m *testing.M) {
	traceExporter = tracetest.NewInMemoryExporter()
	tracePublisher = trace.Setup("test", nil, traceExporter)

	code := m.Run()

	os.Exit(code)
}

func TestNewAsyncProducer_Success(t *testing.T) {
	saramaCfg, err := DefaultConfig("test-producer", true)
	require.NoError(t, err)

	ap, err := NewAsyncProducer(brokers, saramaCfg)
	require.NoError(t, err)
	assert.NotNil(t, ap)
}

func TestNewSyncProducer_Success(t *testing.T) {
	saramaCfg, err := DefaultConfig("test-producer", true)
	require.NoError(t, err)

	p, err := NewSyncProducer(brokers, saramaCfg)
	require.NoError(t, err)
	assert.NotNil(t, p)
}

func TestAsyncProducer_SendMessage_Close(t *testing.T) {
	t.Cleanup(func() { traceExporter.Reset() })

	ctx := context.Background()

	shutdownProvider, assertCollectMetrics := test.SetupMetrics(ctx, t)
	defer shutdownProvider()

	saramaCfg, err := DefaultConfig("test-consumer", false)
	require.NoError(t, err)

	ap, err := NewAsyncProducer(brokers, saramaCfg)
	require.NoError(t, err)
	assert.NotNil(t, ap)
	msg := &sarama.ProducerMessage{
		Topic:   clientTopic,
		Value:   sarama.StringEncoder("TEST"),
		Headers: []sarama.RecordHeader{{Key: []byte("123"), Value: []byte("123")}},
	}
	err = ap.Send(context.Background(), msg)
	require.NoError(t, err)
	require.NoError(t, ap.Close())

	// Tracing
	require.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expected := tracetest.SpanStub{
		Name: "send",
		Attributes: []attribute.KeyValue{
			attribute.String("delivery", "async"),
			attribute.String("client", "kafka"),
			attribute.String("topic", "clientTopic"),
		},
	}

	snaps := traceExporter.GetSpans().Snapshots()

	assert.Len(t, snaps, 1)
	assert.Equal(t, expected.Name, snaps[0].Name())
	assert.Equal(t, expected.Attributes, snaps[0].Attributes())

	// Metrics
	_ = assertCollectMetrics(1)
}

func TestSyncProducer_SendMessage_Close(t *testing.T) {
	t.Cleanup(func() {
		traceExporter.Reset()
	})
	saramaCfg, err := DefaultConfig("test-producer", true)
	require.NoError(t, err)

	p, err := NewSyncProducer(brokers, saramaCfg)
	require.NoError(t, err)
	assert.NotNil(t, p)
	msg := &sarama.ProducerMessage{
		Topic: clientTopic,
		Value: sarama.StringEncoder("TEST"),
	}
	partition, offset, err := p.SendMessage(context.Background(), msg)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, partition, int32(0))
	assert.GreaterOrEqual(t, offset, int64(0))
	require.NoError(t, p.Close())

	// Tracing
	require.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expected := tracetest.SpanStub{
		Name: "send",
		Attributes: []attribute.KeyValue{
			attribute.String("delivery", "sync"),
			attribute.String("client", "kafka"),
			attribute.String("topic", "clientTopic"),
		},
	}

	snaps := traceExporter.GetSpans().Snapshots()

	assert.Len(t, snaps, 1)
	assert.Equal(t, expected.Name, snaps[0].Name())
	assert.Equal(t, expected.Attributes, snaps[0].Attributes())
}

func TestSyncProducer_SendMessages_Close(t *testing.T) {
	t.Cleanup(func() {
		traceExporter.Reset()
	})
	saramaCfg, err := DefaultConfig("test-producer", true)
	require.NoError(t, err)

	p, err := NewSyncProducer(brokers, saramaCfg)
	require.NoError(t, err)
	assert.NotNil(t, p)
	msg1 := &sarama.ProducerMessage{
		Topic: clientTopic,
		Value: sarama.StringEncoder("TEST1"),
	}
	msg2 := &sarama.ProducerMessage{
		Topic: clientTopic,
		Value: sarama.StringEncoder("TEST2"),
	}
	err = p.SendBatch(context.Background(), []*sarama.ProducerMessage{msg1, msg2})
	require.NoError(t, err)
	require.NoError(t, p.Close())
	// Tracing
	require.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expected := tracetest.SpanStub{
		Name: "send-batch",
		Attributes: []attribute.KeyValue{
			attribute.String("delivery", "sync"),
			attribute.String("client", "kafka"),
		},
	}

	snaps := traceExporter.GetSpans().Snapshots()

	assert.Len(t, snaps, 1)
	assert.Equal(t, expected.Name, snaps[0].Name())
	assert.Equal(t, expected.Attributes, snaps[0].Attributes())
}
