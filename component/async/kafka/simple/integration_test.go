// +build integration

package simple

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async/kafka"
	dockerKafka "github.com/beatlabs/patron/test/docker/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	topic = "Topic1"
)

func TestMain(m *testing.M) {
	os.Exit(dockerKafka.RunWithKafka(m, 120*time.Second, fmt.Sprintf("%s:1:1", topic)))
}

func TestConsume(t *testing.T) {
	sent := []string{"one", "two", "three"}
	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {

		factory, err := New("test1", topic, dockerKafka.Brokers(), kafka.DecoderJSON(), kafka.Version(sarama.V2_1_0_0.String()),
			kafka.StartFromNewest())
		if err != nil {
			chErr <- err
			return
		}

		received, err := consumeMessages(factory, len(sent))
		if err != nil {
			chErr <- err
			return
		}

		chMessages <- received
	}()

	time.Sleep(10 * time.Second)

	prod, err := dockerKafka.NewProducer()
	require.NoError(t, err)
	for _, message := range sent {
		_, _, err = prod.SendMessage(getProducerMessage(message))
		require.NoError(t, err)
	}

	var received []string

	select {
	case received = <-chMessages:
	case err = <-chErr:
		require.NoError(t, err)
	}

	assert.Equal(t, sent, received)
}

func TestConsume_ClaimMessageError(t *testing.T) {
	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {

		factory, err := New("test1", topic, dockerKafka.Brokers(), kafka.Version(sarama.V2_1_0_0.String()),
			kafka.StartFromNewest())
		if err != nil {
			chErr <- err
			return
		}

		received, err := consumeMessages(factory, 1)
		if err != nil {
			chErr <- err
			return
		}

		chMessages <- received
	}()

	time.Sleep(10 * time.Second)

	prod, err := dockerKafka.NewProducer()
	require.NoError(t, err)
	_, _, err = prod.SendMessage(getProducerMessage("123"))
	require.NoError(t, err)

	select {
	case <-chMessages:
		require.Fail(t, "no messages where expected")
	case err = <-chErr:
		require.EqualError(t, err, "could not determine decoder  failed to determine content type from message headers [] : content type header is missing")
	}
}

func consumeMessages(factory *Factory, expectedMessageCount int) ([]string, error) {

	consumer, err := factory.Create()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = consumer.Close()
	}()

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

func getProducerMessage(message string) *sarama.ProducerMessage {
	return &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(message),
	}
}
