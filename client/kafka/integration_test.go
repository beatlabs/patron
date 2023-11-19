//go:build integration

package kafka

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	clientTopic = "clientTopic"
)

var brokers = []string{"127.0.0.1:9093"}

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
	saramaCfg, err := DefaultProducerSaramaConfig("test-consumer", false)
	require.Nil(t, err)

	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
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
	assert.Len(t, mtr.FinishedSpans(), 1)

	expected := map[string]interface{}{
		"component": "kafka-async-producer",
		"error":     false,
		"span.kind": ext.SpanKindEnum("producer"),
		"topic":     clientTopic,
		"type":      "async",
		"version":   "dev",
	}
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())

	// Metrics
	assert.Equal(t, 1, testutil.CollectAndCount(messageStatus, "client_kafka_producer_message_status"))
}

func TestSyncProducer_SendMessage_Close(t *testing.T) {
	saramaCfg, err := DefaultProducerSaramaConfig("test-producer", true)
	require.NoError(t, err)

	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
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
	assert.Len(t, mtr.FinishedSpans(), 1)

	expected := map[string]interface{}{
		"component": "kafka-sync-producer",
		"error":     false,
		"span.kind": ext.SpanKindEnum("producer"),
		"topic":     clientTopic,
		"type":      "sync",
		"version":   "dev",
	}
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
}

func TestSyncProducer_SendMessages_Close(t *testing.T) {
	saramaCfg, err := DefaultProducerSaramaConfig("test-producer", true)
	require.NoError(t, err)

	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
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
	assert.Len(t, mtr.FinishedSpans(), 2)

	expected := map[string]interface{}{
		"component": "kafka-sync-producer",
		"error":     false,
		"span.kind": ext.SpanKindEnum("producer"),
		"topic":     "batch",
		"type":      "sync",
		"version":   "dev",
	}
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())
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
