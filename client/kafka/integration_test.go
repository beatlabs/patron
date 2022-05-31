//go:build integration
// +build integration

package kafka

import (
	"context"
	"testing"

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
	ap, chErr, err := NewBuilder(brokers).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
}

func TestNewSyncProducer_Success(t *testing.T) {
	p, err := NewBuilder(brokers).CreateSync()
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestAsyncProducer_SendMessage_Close(t *testing.T) {
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
	ap, chErr, err := NewBuilder(brokers).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
	msg := NewMessage(clientTopic, "TEST")
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
	assert.Equal(t, 1, testutil.CollectAndCount(messageStatus, "component_kafka_producer_message_status"))
}

func TestSyncProducer_SendMessage_Close(t *testing.T) {
	mtr := mocktracer.New()
	defer mtr.Reset()
	opentracing.SetGlobalTracer(mtr)
	p, err := NewBuilder(brokers).CreateSync()
	require.NoError(t, err)
	assert.NotNil(t, p)
	msg := NewMessage(clientTopic, "TEST")
	err = p.Send(context.Background(), msg)
	assert.NoError(t, err)
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
	ap, chErr, err := NewBuilder(brokers).CreateAsync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotNil(t, chErr)
	assert.NotEmpty(t, ap.ActiveBrokers())
	assert.NoError(t, ap.Close())
}

func TestSyncProducerActiveBrokers(t *testing.T) {
	ap, err := NewBuilder(brokers).CreateSync()
	assert.NoError(t, err)
	assert.NotNil(t, ap)
	assert.NotEmpty(t, ap.ActiveBrokers())
	assert.NoError(t, ap.Close())
}
