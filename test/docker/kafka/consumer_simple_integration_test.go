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

func TestSimpleConsume_WithDurationOffset(t *testing.T) {
	now := time.Now()
	sent := createTimestampPayload(
		now.Add(-10*time.Hour),
		now.Add(-5*time.Hour),
		now.Add(-3*time.Hour),
		now.Add(-2*time.Hour),
		now.Add(-1*time.Hour),
	)

	messages := make([]*sarama.ProducerMessage, 0)
	for _, val := range sent {
		messages = append(messages, getProducerMessage(simpleTopic3, val))
	}

	err := sendMessages(messages...)
	require.NoError(t, err)

	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {
		factory, err := simple.New("test1", simpleTopic3, Brokers(), kafka.DecoderJSON(), kafka.Version(sarama.V2_1_0_0.String()),
			kafka.StartFromNewest(), simple.WithDurationOffset(4*time.Hour, timestampExtractor))
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

		received, err := consumeMessages(consumer, 3)
		if err != nil {
			chErr <- err
			return
		}

		chMessages <- received
	}()

	time.Sleep(5 * time.Second)

	var received []string

	select {
	case received = <-chMessages:
	case err = <-chErr:
		require.NoError(t, err)
	}

	assert.Equal(t, sent[2:], received)
}

func TestSimpleConsume_WithNotificationOnceReachingLatestOffset(t *testing.T) {
	messages := make([]*sarama.ProducerMessage, 0)
	numberOfMessages := 10
	for i := 0; i < numberOfMessages; i++ {
		messages = append(messages, getProducerMessage(simpleTopic4, "foo"))
	}

	err := sendMessages(messages...)
	require.NoError(t, err)

	chMessages := make(chan []string)
	chErr := make(chan error)
	chNotif := make(chan struct{})
	go func() {
		factory, err := simple.New("test4", simpleTopic4, Brokers(), kafka.DecoderJSON(), kafka.Version(sarama.V2_1_0_0.String()),
			kafka.StartFromOldest(), simple.WithNotificationOnceReachingLatestOffset(chNotif))
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

		received, err := consumeMessages(consumer, numberOfMessages)
		if err != nil {
			chErr <- err
			return
		}

		chMessages <- received
	}()

	time.Sleep(5 * time.Second)

	select {
	case <-chMessages:
		break
	case err = <-chErr:
		require.NoError(t, err)
	}

	// At this stage, we have received all the expected messages.
	// We should also check that the notification channel is also eventually closed.
	select {
	case <-time.After(time.Second):
		assert.FailNow(t, "notification channel not closed")
	case _, open := <-chNotif:
		assert.False(t, open)
	}
}

func createTimestampPayload(timestamps ...time.Time) []string {
	payloads := make([]string, len(timestamps))
	for i, timestamp := range timestamps {
		payloads[i] = timestamp.Format(time.RFC3339)
	}
	return payloads
}

func timestampExtractor(msg *sarama.ConsumerMessage) (time.Time, error) {
	return time.Parse(time.RFC3339, string(msg.Value))
}
