// +build integration

package amqp

import (
	"context"
	"testing"

	v2 "github.com/beatlabs/patron/client/amqp/v2"
	patronamqp "github.com/beatlabs/patron/component/amqp"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	defer mtr.Reset()

	ctx, cnl := context.WithCancel(context.Background())

	pub, err := v2.New(runtime.getEndpoint())
	require.NoError(t, err)

	sent := []string{"one", "two"}

	err = pub.Publish(ctx, "", rabbitMQQueue, false, false,
		amqp.Publishing{ContentType: "text/plain", Body: []byte(sent[0])})
	require.NoError(t, err)

	err = pub.Publish(ctx, "", rabbitMQQueue, false, false,
		amqp.Publishing{ContentType: "text/plain", Body: []byte(sent[1])})
	require.NoError(t, err)
	mtr.Reset()

	chReceived := make(chan []string)
	received := make([]string, 0)
	count := 0

	procFunc := func(_ context.Context, b patronamqp.Batch) {
		for _, msg := range b.Messages() {
			received = append(received, string(msg.Body()))
			assert.NoError(t, msg.ACK())
		}

		count += len(b.Messages())
		if count == len(sent) {
			chReceived <- received
		}
	}

	cmp, err := patronamqp.New(runtime.getEndpoint(), rabbitMQQueue, procFunc)
	require.NoError(t, err)

	chDone := make(chan struct{})

	go func() {
		require.NoError(t, cmp.Run(ctx))
		chDone <- struct{}{}
	}()

	got := <-chReceived
	cnl()
	assert.ElementsMatch(t, sent, got)
	assert.Len(t, mtr.FinishedSpans(), 2)
	<-chDone
}
