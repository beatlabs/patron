//go:build integration
// +build integration

package v2

import (
	"context"
	"testing"

	testamqp "github.com/beatlabs/patron/test/amqp"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	endpoint = "amqp://user:bitnami@localhost:5672/"
	queue    = "rmq-test-v2-pub-queue"
)

func TestRun(t *testing.T) {
	mtr := mocktracer.New()
	opentracing.SetGlobalTracer(mtr)
	t.Cleanup(func() { mtr.Reset() })

	require.NoError(t, testamqp.CreateQueue(endpoint, queue))

	pub, err := New(endpoint)
	require.NoError(t, err)

	sent := "sent"

	err = pub.Publish(context.Background(), "", queue, false, false,
		amqp.Publishing{ContentType: "text/plain", Body: []byte(sent)})
	require.NoError(t, err)

	expected := map[string]interface{}{
		"component": "amqp-publisher",
		"error":     false,
		"exchange":  "",
		"span.kind": ext.SpanKindEnum("producer"),
		"version":   "dev",
	}

	assert.Len(t, mtr.FinishedSpans(), 1)
	assert.Equal(t, expected, mtr.FinishedSpans()[0].Tags())

	// Metrics
	assert.Equal(t, 1, testutil.CollectAndCount(publishDurationMetrics, "client_amqp_publish_duration_seconds"))

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
	assert.NoError(t, channel.Close())
	assert.NoError(t, conn.Close())
}
