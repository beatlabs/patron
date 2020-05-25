// +build integration

package kafka

import (
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async/kafka"
	"github.com/beatlabs/patron/component/async/kafka/simple"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	simpleTopic1 = "simpleTopic1"
	simpleTopic2 = "simpleTopic2"
)

func TestSimpleConsume(t *testing.T) {
	sent := []string{"one", "two", "three"}
	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {

		factory, err := simple.New("test1", simpleTopic1, Brokers(), kafka.DecoderJSON(), kafka.Version(sarama.V2_1_0_0.String()),
			kafka.StartFromNewest())
		if err != nil {
			chErr <- err
			return
		}

		consumer, err := factory.Create()
		if err != nil {
			chErr <- err
			return
		}
		defer func() {
			_ = consumer.Close()
		}()

		received, err := consumeMessages(consumer, len(sent))
		if err != nil {
			chErr <- err
			return
		}

		chMessages <- received
	}()

	time.Sleep(5 * time.Second)

	messages := make([]*sarama.ProducerMessage, 0, len(sent))
	for _, val := range sent {
		messages = append(messages, getProducerMessage(simpleTopic1, val))
	}

	err := sendMessages(messages...)
	require.NoError(t, err)

	var received []string

	select {
	case received = <-chMessages:
	case err = <-chErr:
		require.NoError(t, err)
	}

	assert.Equal(t, sent, received)
}

func TestSimpleConsume_ClaimMessageError(t *testing.T) {
	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {

		factory, err := simple.New("test1", simpleTopic2, Brokers(), kafka.Version(sarama.V2_1_0_0.String()),
			kafka.StartFromNewest())
		if err != nil {
			chErr <- err
			return
		}

		consumer, err := factory.Create()
		if err != nil {
			chErr <- err
			return
		}
		defer func() {
			_ = consumer.Close()
		}()

		received, err := consumeMessages(consumer, 1)
		if err != nil {
			chErr <- err
			return
		}

		chMessages <- received
	}()

	time.Sleep(5 * time.Second)

	err := sendMessages(getProducerMessage(simpleTopic2, "123"))
	require.NoError(t, err)

	select {
	case <-chMessages:
		require.Fail(t, "no messages where expected")
	case err = <-chErr:
		require.EqualError(t, err, "could not determine decoder  failed to determine content type from message headers [] : content type header is missing")
	}
}
