//go:build integration
// +build integration

package simple

import (
	"context"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async/kafka"
	kafkacmp "github.com/beatlabs/patron/component/kafka"
	testkafka "github.com/beatlabs/patron/test/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	simpleTopic1 = "simpleTopic1"
	simpleTopic2 = "simpleTopic2"
	simpleTopic3 = "simpleTopic3"
	simpleTopic4 = "simpleTopic4"
	simpleTopic5 = "simpleTopic5"
	simpleTopic6 = "simpleTopic6"
	simpleTopic7 = "simpleTopic7"
	broker       = "127.0.0.1:9093"
)

func TestSimpleConsume(t *testing.T) {
	require.NoError(t, testkafka.CreateTopics(broker, simpleTopic1))
	sent := []string{"one", "two", "three"}
	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {
		saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig("test-simple-consumer", true)
		require.NoError(t, err)

		factory, err := New("test1", simpleTopic1, []string{broker}, saramaCfg, kafka.WithDecoderJSON(), kafka.WithVersion(sarama.V2_1_0_0.String()),
			kafka.WithStartFromNewest())
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

		received, err := testkafka.AsyncConsumeMessages(consumer, len(sent))
		if err != nil {
			chErr <- err
			return
		}

		chMessages <- received
	}()

	time.Sleep(5 * time.Second)

	messages := make([]*sarama.ProducerMessage, 0, len(sent))
	for _, val := range sent {
		messages = append(messages, testkafka.CreateProducerMessage(simpleTopic1, val))
	}

	err := testkafka.SendMessages(broker, messages...)
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
	require.NoError(t, testkafka.CreateTopics(broker, simpleTopic2))
	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {
		saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig("test-simple-consumer-claim", true)
		require.NoError(t, err)

		factory, err := New("test1", simpleTopic2, []string{broker}, saramaCfg, kafka.WithVersion(sarama.V2_1_0_0.String()),
			kafka.WithStartFromNewest())
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

		received, err := testkafka.AsyncConsumeMessages(consumer, 1)
		if err != nil {
			chErr <- err
			return
		}

		chMessages <- received
	}()

	time.Sleep(5 * time.Second)

	err := testkafka.SendMessages(broker, testkafka.CreateProducerMessage(simpleTopic2, "123"))
	require.NoError(t, err)

	select {
	case <-chMessages:
		require.Fail(t, "no messages where expected")
	case err = <-chErr:
		require.EqualError(t, err, "could not determine decoder  failed to determine content type from message headers [] : content type header is missing")
	}
}

func TestSimpleConsume_WithDurationOffset(t *testing.T) {
	require.NoError(t, testkafka.CreateTopics(broker, simpleTopic3))
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
		messages = append(messages, testkafka.CreateProducerMessage(simpleTopic3, val))
	}

	err := testkafka.SendMessages(broker, messages...)
	require.NoError(t, err)

	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {
		saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig("test-simple-consumer-w-duration", true)
		require.NoError(t, err)

		factory, err := New("test1", simpleTopic3, []string{broker}, saramaCfg, kafka.WithDecoderJSON(), kafka.WithVersion(sarama.V2_1_0_0.String()),
			kafka.WithStartFromNewest(), WithDurationOffset(4*time.Hour, timestampExtractor))
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

		received, err := testkafka.AsyncConsumeMessages(consumer, 3)
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

func TestSimpleConsume_WithTimestampOffset(t *testing.T) {
	require.NoError(t, testkafka.CreateTopics(broker, simpleTopic6))
	now := time.Now()
	times := []time.Time{
		now.Add(-10 * time.Hour),
		now.Add(-5 * time.Hour),
		now.Add(-3 * time.Hour),
		now.Add(-2 * time.Hour),
		now.Add(-1 * time.Hour),
	}
	sent := createTimestampPayload(times...)

	messages := make([]*sarama.ProducerMessage, 0)
	for i, tm := range times {
		val := sent[i]
		msg := testkafka.CreateProducerMessage(simpleTopic6, val)
		msg.Timestamp = tm
		messages = append(messages, msg)
	}

	err := testkafka.SendMessages(broker, messages...)
	require.NoError(t, err)

	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {
		saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig("test-simple-consumer-w-timestamp", true)
		require.NoError(t, err)

		factory, err := New("test1", simpleTopic6, []string{broker}, saramaCfg, kafka.WithDecoderJSON(), kafka.WithVersion(sarama.V2_1_0_0.String()),
			kafka.WithStartFromNewest(), WithTimestampOffset(4*time.Hour))
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

		received, err := testkafka.AsyncConsumeMessages(consumer, 3)
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
	require.NoError(t, testkafka.CreateTopics(broker, simpleTopic4))
	messages := make([]*sarama.ProducerMessage, 0)
	numberOfMessages := 10
	for i := 0; i < numberOfMessages; i++ {
		messages = append(messages, testkafka.CreateProducerMessage(simpleTopic4, "foo"))
	}

	err := testkafka.SendMessages(broker, messages...)
	require.NoError(t, err)

	chMessages := make(chan []string)
	chErr := make(chan error)
	chNotif := make(chan struct{})
	go func() {
		saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig("test-simple-consumer-w-notif", true)
		require.NoError(t, err)

		factory, err := New("test4", simpleTopic4, []string{broker}, saramaCfg, kafka.WithDecoderJSON(), kafka.WithVersion(sarama.V2_1_0_0.String()),
			kafka.WithStartFromOldest(), WithNotificationOnceReachingLatestOffset(chNotif))
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

		received, err := testkafka.AsyncConsumeMessages(consumer, numberOfMessages)
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

func TestSimpleConsume_WithNotificationOnceReachingLatestOffset_NoMessages(t *testing.T) {
	require.NoError(t, testkafka.CreateTopics(broker, simpleTopic5))
	chErr := make(chan error)
	chNotif := make(chan struct{})
	go func() {
		saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig("test-simple-consumer", true)
		require.NoError(t, err)

		factory, err := New("test5", simpleTopic5, []string{broker}, saramaCfg, kafka.WithDecoderJSON(), kafka.WithVersion(sarama.V2_1_0_0.String()),
			kafka.WithStartFromOldest(), WithNotificationOnceReachingLatestOffset(chNotif))
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

		ctx, cnl := context.WithCancel(context.Background())
		defer cnl()

		_, _, err = consumer.Consume(ctx)
		if err != nil {
			chErr <- err
		}
	}()

	// At this stage, we have received all the expected messages.
	// We should also check that the notification channel is also eventually closed.
	select {
	case err := <-chErr:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		assert.FailNow(t, "notification channel not closed")
	case _, open := <-chNotif:
		assert.False(t, open)
	}
}

func TestSimpleConsume_WithNotificationOnceReachingLatestOffset_WithTimestampOffset_RarelyUpdatedTopic(t *testing.T) {
	require.NoError(t, testkafka.CreateTopics(broker, simpleTopic7))
	// Messages with old timestamps
	now := time.Now()
	times := []time.Time{
		now.Add(-10 * time.Hour),
		now.Add(-8 * time.Hour),
		now.Add(-5 * time.Hour),
	}
	sent := createTimestampPayload(times...)

	messages := make([]*sarama.ProducerMessage, 0)
	for i, tm := range times {
		val := sent[i]
		msg := testkafka.CreateProducerMessage(simpleTopic7, val)
		msg.Timestamp = tm
		messages = append(messages, msg)
	}

	err := testkafka.SendMessages(broker, messages...)
	require.NoError(t, err)

	chErr := make(chan error)
	chNotif := make(chan struct{})
	go func() {
		saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig("test-simple-consumer", true)
		require.NoError(t, err)

		factory, err := New("test7", simpleTopic7, []string{broker}, saramaCfg, kafka.WithDecoderJSON(), kafka.WithVersion(sarama.V2_1_0_0.String()),
			WithTimestampOffset(4*time.Hour), WithNotificationOnceReachingLatestOffset(chNotif))
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

		ctx, cnl := context.WithCancel(context.Background())
		defer cnl()

		_, _, err = consumer.Consume(ctx)
		if err != nil {
			chErr <- err
		}
	}()

	// At this stage, we have received all the expected messages.
	// We should also check that the notification channel is also eventually closed.
	select {
	case err := <-chErr:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
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
