//go:build integration
// +build integration

package kafka

import (
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async/kafka"
	"github.com/beatlabs/patron/component/async/kafka/group"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupConsume(t *testing.T) {
	sent := []string{"one", "two", "three"}
	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {

		factory, err := group.New("test1", uuid.New().String(), []string{groupTopic1}, Brokers(), kafka.DecoderJSON(),
			kafka.Version(sarama.V2_1_0_0.String()), kafka.StartFromNewest())
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
		messages = append(messages, getProducerMessage(groupTopic1, val))
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

func TestGroupConsume_ClaimMessageError(t *testing.T) {
	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {

		// Consumer will error out in ClaimMessage as no DecoderFunc has been set
		factory, err := group.New("test1", uuid.New().String(), []string{groupTopic2}, Brokers(),
			kafka.Version(sarama.V2_1_0_0.String()), kafka.StartFromNewest())
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

	err := sendMessages(getProducerMessage(groupTopic2, "321"))
	require.NoError(t, err)

	select {
	case <-chMessages:
		require.Fail(t, "no messages were expected")
	case err = <-chErr:
		require.EqualError(t, err, "kafka: error while consuming groupTopic2/0: "+
			"could not determine decoder  failed to determine content type from message headers [] : "+
			"content type header is missing")
	}
}
