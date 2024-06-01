//go:build integration

package kafka

import (
	"context"
	"os"
	"testing"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron/observability/trace"
	"github.com/prometheus/client_golang/prometheus/testutil"
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
	var err error
	tracePublisher, err = trace.Setup("test", nil, traceExporter)
	if err != nil {
		os.Exit(1)
	}

	code := m.Run()

	os.Exit(code)
}

func TestNewAsyncProducer_Success(t *testing.T) {
	saramaCfg, err := DefaultProducerSaramaConfig("test-producer", true)
	require.Nil(t, err)

	ap, chErr, err := New(brokers, saramaCfg).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
}

func TestNewSyncProducer_Success(t *testing.T) {
	saramaCfg, err := DefaultProducerSaramaConfig("test-producer", true)
	require.Nil(t, err)

	p, err := New(brokers, saramaCfg).Create()
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestAsyncProducer_SendMessage_Close(t *testing.T) {
	t.Cleanup(func() {
		traceExporter.Reset()
	})
	saramaCfg, err := DefaultProducerSaramaConfig("test-consumer", false)
	require.Nil(t, err)

	ap, chErr, err := New(brokers, saramaCfg).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
	msg := &sarama.ProducerMessage{
		Topic:   clientTopic,
		Value:   sarama.StringEncoder("TEST"),
		Headers: []sarama.RecordHeader{{Key: []byte("123"), Value: []byte("123")}},
	}
	err = ap.Send(context.Background(), msg)
	assert.NoError(t, err)
	assert.NoError(t, ap.Close())

	// Tracing
	assert.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expected := tracetest.SpanStub{
		Name: "send",
		Attributes: []attribute.KeyValue{
			attribute.String("delivery", "async"),
			attribute.String("component", "kafka"),
			attribute.String("topic", "clientTopic"),
		},
	}

	snaps := traceExporter.GetSpans().Snapshots()

	assert.Len(t, snaps, 1)
	assert.Equal(t, expected.Name, snaps[0].Name())
	assert.Equal(t, expected.Attributes, snaps[0].Attributes())

	// Metrics
	assert.Equal(t, 1, testutil.CollectAndCount(messageStatus, "client_kafka_producer_message_status"))
}

func TestSyncProducer_SendMessage_Close(t *testing.T) {
	t.Cleanup(func() {
		traceExporter.Reset()
	})
	saramaCfg, err := DefaultProducerSaramaConfig("test-producer", true)
	require.NoError(t, err)

	p, err := New(brokers, saramaCfg).Create()
	require.NoError(t, err)
	assert.NotNil(t, p)
	msg := &sarama.ProducerMessage{
		Topic: clientTopic,
		Value: sarama.StringEncoder("TEST"),
	}
	partition, offset, err := p.Send(context.Background(), msg)
	assert.NoError(t, err)
	assert.True(t, partition >= 0)
	assert.True(t, offset >= 0)
	assert.NoError(t, p.Close())

	// Tracing
	assert.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expected := tracetest.SpanStub{
		Name: "send",
		Attributes: []attribute.KeyValue{
			attribute.String("delivery", "sync"),
			attribute.String("component", "kafka"),
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
	saramaCfg, err := DefaultProducerSaramaConfig("test-producer", true)
	require.NoError(t, err)

	p, err := New(brokers, saramaCfg).Create()
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
	assert.NoError(t, err)
	assert.NoError(t, p.Close())
	// Tracing
	assert.NoError(t, tracePublisher.ForceFlush(context.Background()))

	expected := tracetest.SpanStub{
		Name: "send-batch",
		Attributes: []attribute.KeyValue{
			attribute.String("delivery", "sync"),
			attribute.String("component", "kafka"),
		},
	}

	snaps := traceExporter.GetSpans().Snapshots()

	assert.Len(t, snaps, 1)
	assert.Equal(t, expected.Name, snaps[0].Name())
	assert.Equal(t, expected.Attributes, snaps[0].Attributes())
}

func TestAsyncProducerActiveBrokers(t *testing.T) {
	saramaCfg, err := DefaultProducerSaramaConfig("test-producer", true)
	require.NoError(t, err)

	ap, chErr, err := New(brokers, saramaCfg).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
	assert.NotEmpty(t, ap.ActiveBrokers())
	assert.NoError(t, ap.Close())
}

func TestSyncProducerActiveBrokers(t *testing.T) {
	saramaCfg, err := DefaultProducerSaramaConfig("test-producer", true)
	require.NoError(t, err)

	ap, err := New(brokers, saramaCfg).Create()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotEmpty(t, ap.ActiveBrokers())
	assert.NoError(t, ap.Close())
}
