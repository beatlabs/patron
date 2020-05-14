// +build integration

package simple

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async/kafka/test"
	"github.com/stretchr/testify/require"
)

const (
	topic = "Topic1"
)

func TestMain(m *testing.M) {
	os.Exit(test.RunWithKafka(m, 120*time.Second, fmt.Sprintf("%s:1:1", topic)))
}

func TestConsume(t *testing.T) {
	// Send messages
	sent := []string{"one", "two", "three"}
	prod, err := test.NewProducer()
	require.NoError(t, err)
	err = prod.SendMessages(getProducerMessages(sent))
	require.NoError(t, err)

	// consume messages
	factory, err := New("test1", topic, test.Brokers())
	require.NoError(t, err)
	consumer, err := factory.Create()
	require.NoError(t, err)
	ctx, cnl := context.WithCancel(context.Background())
	defer cnl()

	ch, chErr, err := consumer.Consume(ctx)
	require.NoError(t, err)

	received := make([]string, 0, len(sent))

	for {
		select {
		case msg := <-ch:
		case err = <-chErr:
			cnl()
			assert.NoError(t, err)
		}
	}
	// verify
}

func getProducerMessages(messages []string) []*sarama.ProducerMessage {
	mm := make([]*sarama.ProducerMessage, 0, len(messages))
	for _, message := range messages {
		mm = append(mm, &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.StringEncoder(message),
		})
	}
	return mm
}
