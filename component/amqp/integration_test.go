//go:build integration
// +build integration

package amqp

import (
	"context"
	"testing"
	"time"

	v2 "github.com/beatlabs/patron/client/amqp/v2"
	"github.com/beatlabs/patron/correlation"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	endpoint      = "amqp://user:bitnami@localhost:5672/"
	rabbitMQQueue = "rmq-test-queue"
)

func TestRun(t *testing.T) {
	require.NoError(t, createQueue())
	t.Cleanup(func() { mtr.Reset() })

	ctx, cnl := context.WithCancel(context.Background())

	pub, err := v2.New(endpoint)
	require.NoError(t, err)

	sent := []string{"one", "two"}

	reqCtx := correlation.ContextWithID(ctx, "123")

	err = pub.Publish(reqCtx, "", rabbitMQQueue, false, false,
		amqp.Publishing{ContentType: "text/plain", Body: []byte(sent[0])})
	require.NoError(t, err)

	err = pub.Publish(reqCtx, "", rabbitMQQueue, false, false,
		amqp.Publishing{ContentType: "text/plain", Body: []byte(sent[1])})
	require.NoError(t, err)
	mtr.Reset()

	chReceived := make(chan []string)
	received := make([]string, 0)
	count := 0

	procFunc := func(_ context.Context, b Batch) {
		for _, msg := range b.Messages() {
			received = append(received, string(msg.Body()))
			assert.NoError(t, msg.ACK())
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
		require.NoError(t, cmp.Run(ctx))
		chDone <- struct{}{}
	}()

	got := <-chReceived
	cnl()

	<-chDone

	assert.ElementsMatch(t, sent, got)
	assert.Len(t, mtr.FinishedSpans(), 2)

	expectedTags := map[string]interface{}{
		"component":     "amqp-consumer",
		"correlationID": "123",
		"error":         false,
		"queue":         "rmq-test-queue",
		"span.kind":     ext.SpanKindEnum("consumer"),
		"version":       "dev",
	}

	for _, span := range mtr.FinishedSpans() {
		assert.Equal(t, expectedTags, span.Tags())
	}

	assert.Equal(t, 1, testutil.CollectAndCount(messageAge, "component_amqp_message_age"))
	assert.Equal(t, 2, testutil.CollectAndCount(messageCounterVec, "component_amqp_message_counter"))
	assert.GreaterOrEqual(t, testutil.CollectAndCount(queueSize, "component_amqp_queue_size"), 0)
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
