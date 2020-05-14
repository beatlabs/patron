// +build integration

package simple

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/beatlabs/patron/component/async/kafka"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async/kafka/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	topic = "Topic1"
)

func TestMain(m *testing.M) {
	os.Exit(test.RunWithKafka(m, 120*time.Second, fmt.Sprintf("%s:1:1", topic)))
}

func TestConsume(t *testing.T) {
	sent := []string{"one", "two", "three"}
	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {

		received, err := consumeMessages(len(sent))
		if err != nil {
			chErr <- err
			return
		}

		chMessages <- received
	}()

	// Send messages
	prod, err := test.NewProducer()
	require.NoError(t, err)
	for _, message := range getProducerMessages(sent) {
		_, _, err = prod.SendMessage(message)
		require.NoError(t, err)
	}

	var received []string

	select {
	case received = <-chMessages:
	case err = <-chErr:
		require.NoError(t, err)
	}

	// verify
	assert.Equal(t, sent, received)
}

func consumeMessages(expectedMessageCount int) ([]string, error) {

	// consume messages
	factory, err := New("test1", topic, test.Brokers(), kafka.DecoderJSON(), kafka.Version(sarama.V2_1_0_0.String()),
		kafka.StartFromNewest())
	if err != nil {
		return nil, err
	}
	consumer, err := factory.Create()
	if err != nil {
		return nil, err
	}

	ctx, cnl := context.WithCancel(context.Background())
	defer cnl()

	ch, chErr, err := consumer.Consume(ctx)
	if err != nil {
		return nil, err
	}

	received := make([]string, 0, expectedMessageCount)

	for {
		select {
		case msg := <-ch:
			received = append(received, string(msg.Payload()))
			expectedMessageCount--
			if expectedMessageCount == 0 {
				return received, nil
			}
		case err := <-chErr:
			return nil, err
		}
	}
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
