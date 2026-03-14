//go:build integration

package kafka

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/beatlabs/patron/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/goleak"
)

const (
	clientTopic = "clientTopic"
)

var brokers = []string{"127.0.0.1:9092"}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("google.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/metric.(*PeriodicReader).run"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"),
	)
}

func TestNewProducer_Success(t *testing.T) {
	host, err := os.Hostname()
	require.NoError(t, err)

	p, err := New(brokers, kgo.ClientID(fmt.Sprintf("%s-%s", host, "test-producer")), kgo.RequiredAcks(kgo.AllISRAcks()))
	require.NoError(t, err)
	assert.NotNil(t, p)
	p.Close()
}

func TestProducer_SendAsync_Close(t *testing.T) {
	require.NoError(t, createTopics(brokers[0], clientTopic))

	ctx := context.Background()

	shutdownProvider, assertCollectMetrics := test.SetupMetrics(ctx, t)
	defer shutdownProvider()

	p, err := New(brokers, kgo.DisableIdempotentWrite())
	require.NoError(t, err)
	assert.NotNil(t, p)
	msg := &kgo.Record{
		Topic:   clientTopic,
		Value:   []byte("TEST"),
		Headers: []kgo.RecordHeader{{Key: "123", Value: []byte("123")}},
	}
	chErr := make(chan error, 1)
	p.SendAsync(context.Background(), msg, chErr)
	p.Close()

	select {
	case err := <-chErr:
		require.NoError(t, err)
	default:
	}

	// Metrics — kotel handles tracing; we only assert our custom publish count metric.
	_ = assertCollectMetrics(1)
}

func TestProducer_Send_Close(t *testing.T) {
	require.NoError(t, createTopics(brokers[0], clientTopic))

	p, err := New(brokers, kgo.RequiredAcks(kgo.AllISRAcks()))
	require.NoError(t, err)
	assert.NotNil(t, p)

	msg := &kgo.Record{
		Topic: clientTopic,
		Value: []byte("TEST"),
	}
	produced, err := p.Send(context.Background(), msg)
	require.NoError(t, err)
	require.Len(t, produced, 1)
	assert.GreaterOrEqual(t, produced[0].Partition, int32(0))
	assert.GreaterOrEqual(t, produced[0].Offset, int64(0))
	p.Close()
}

func TestProducer_SendBatch_Close(t *testing.T) {
	require.NoError(t, createTopics(brokers[0], clientTopic))

	p, err := New(brokers, kgo.RequiredAcks(kgo.AllISRAcks()))
	require.NoError(t, err)
	assert.NotNil(t, p)

	msg1 := &kgo.Record{
		Topic: clientTopic,
		Value: []byte("TEST1"),
	}
	msg2 := &kgo.Record{
		Topic: clientTopic,
		Value: []byte("TEST2"),
	}
	produced, err := p.Send(context.Background(), msg1, msg2)
	require.NoError(t, err)
	require.Len(t, produced, 2)
	p.Close()
}

func createTopics(broker string, topics ...string) error {
	cl, err := kgo.NewClient(kgo.SeedBrokers(broker))
	if err != nil {
		return err
	}
	defer cl.Close()

	adm := kadm.NewClient(cl)

	for _, topic := range topics {
		_, _ = adm.DeleteTopics(context.Background(), topic)
	}

	time.Sleep(100 * time.Millisecond)

	resp, err := adm.CreateTopics(context.Background(), 1, 1, nil, topics...)
	if err != nil {
		return err
	}
	for _, topic := range resp {
		if topic.Err != nil {
			return fmt.Errorf("failed to create topic %s: %w", topic.Topic, topic.Err)
		}
	}

	return nil
}
