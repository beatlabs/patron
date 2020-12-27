// +build integration

package kafka

import (
	"context"
	"testing"

	"github.com/Shopify/sarama"
	v2 "github.com/beatlabs/patron/client/kafka/v2"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	clientTopic = "clientTopic"
)

func TestNewAsyncProducer_Success(t *testing.T) {
	ap, chErr, err := v2.New(Brokers()).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
}

func TestNewSyncProducer_Success(t *testing.T) {
	p, err := v2.New(Brokers()).Create()
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestAsyncProducer_SendMessage_Close(t *testing.T) {
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
	ap, chErr, err := v2.New(Brokers()).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
	msg := &sarama.ProducerMessage{
		Topic: clientTopic,
		Value: sarama.StringEncoder("TEST"),
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
}

func TestSyncProducer_SendMessage_Close(t *testing.T) {
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
	p, err := v2.New(Brokers()).Create()
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

func TestAsyncProducerActiveBrokers(t *testing.T) {
	ap, chErr, err := v2.New(Brokers()).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
	assert.NotEmpty(t, ap.ActiveBrokers())
	assert.NoError(t, ap.Close())
}

func TestSyncProducerActiveBrokers(t *testing.T) {
	ap, err := v2.New(Brokers()).Create()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotEmpty(t, ap.ActiveBrokers())
	assert.NoError(t, ap.Close())
}
