//go:build integration
// +build integration

package kafka

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/component/kafka/simple"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKafkaComponentSimple_Success(t *testing.T) {
	// Test parameters
	numOfMessagesToSend := 100

	// Set up the kafka component
	actualSuccessfulMessages := make([]string, 0)
	var consumerWG sync.WaitGroup
	consumerWG.Add(numOfMessagesToSend)
	processorFunc := func(batch kafka.Batch) error {
		for _, msg := range batch.Messages() {
			var msgContent string
			err := decodeString(msg.Message().Value, &msgContent)
			assert.NoError(t, err)
			actualSuccessfulMessages = append(actualSuccessfulMessages, msgContent)
			consumerWG.Done()
		}
		return nil
	}
	component := newSimpleComponent(t, successTopic3, 3, 10, processorFunc)

	// Run Patron with the kafka component
	patronContext, patronCancel := context.WithCancel(context.Background())
	var patronWG sync.WaitGroup
	patronWG.Add(1)
	go func() {
		svc, err := patron.New(successTopic3, "0", patron.LogFields(map[string]interface{}{"test": successTopic3}))
		require.NoError(t, err)
		err = svc.WithComponents(component).Run(patronContext)
		require.NoError(t, err)
		patronWG.Done()
	}()

	// Send messages to the kafka topic
	var producerWG sync.WaitGroup
	producerWG.Add(1)
	go func() {
		producer, err := NewProducer()
		require.NoError(t, err)
		for i := 1; i <= numOfMessagesToSend; i++ {
			_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: successTopic3, Value: sarama.StringEncoder(strconv.Itoa(i))})
			require.NoError(t, err)
		}
		producerWG.Done()
	}()

	// Wait for both consumer and producer to finish processing all the messages.
	producerWG.Wait()
	consumerWG.Wait()

	// Verify all messages were processed in the right order
	expectedMessages := make([]string, numOfMessagesToSend)
	for i := 0; i < numOfMessagesToSend; i++ {
		expectedMessages[i] = strconv.Itoa(i + 1)
	}
	assert.Equal(t, expectedMessages, actualSuccessfulMessages)

	// Shutdown Patron and wait for it to finish
	patronCancel()
	patronWG.Wait()
}

func TestKafkaComponentSimple_DurationOffset(t *testing.T) {
	// Test parameters
	numOfMessagesToSend := 5

	// Set up the kafka component
	actualSuccessfulMessages := make([]string, 0)
	var consumerWG sync.WaitGroup
	consumerWG.Add(numOfMessagesToSend)
	processorFunc := func(batch kafka.Batch) error {
		for _, msg := range batch.Messages() {
			var msgContent string
			err := decodeString(msg.Message().Value, &msgContent)
			assert.NoError(t, err)
			actualSuccessfulMessages = append(actualSuccessfulMessages, msgContent)
			consumerWG.Done()
		}
		return nil
	}
	component := newSimpleComponent(t, successTopic3, 3, 10, processorFunc, simple.DurationOffset(4*time.Hour, timestampExtractor))

	// Run Patron with the kafka component
	patronContext, patronCancel := context.WithCancel(context.Background())
	var patronWG sync.WaitGroup
	patronWG.Add(1)
	go func() {
		svc, err := patron.New(successTopic3, "0", patron.LogFields(map[string]interface{}{"test": successTopic3}))
		require.NoError(t, err)
		err = svc.WithComponents(component).Run(patronContext)
		require.NoError(t, err)
		patronWG.Done()
	}()

	// Send messages to the kafka topic
	var producerWG sync.WaitGroup
	producerWG.Add(1)
	now := time.Now()
	messages := createTimestampPayload(
		now.Add(-10*time.Hour),
		now.Add(-5*time.Hour),
		now.Add(-3*time.Hour),
		now.Add(-2*time.Hour),
		now.Add(-1*time.Hour),
	)
	go func() {
		producer, err := NewProducer()
		require.NoError(t, err)
		for _, val := range messages {
			_, _, err := producer.SendMessage(getProducerMessage(simpleTopic3, val))
			require.NoError(t, err)
		}
		producerWG.Done()
	}()

	// Wait for both consumer and producer to finish processing all the messages.
	producerWG.Wait()
	consumerWG.Wait()

	// Verify all messages were processed in the right order
	assert.Equal(t, messages[2:], actualSuccessfulMessages)

	// Shutdown Patron and wait for it to finish
	patronCancel()
	patronWG.Wait()
}

func TestKafkaComponentSimple_NotificationOnceReachingLatestOffset(t *testing.T) {
	// Test parameters
	numOfMessagesToSend := 100

	// Set up the kafka component
	actualSuccessfulMessages := make([]string, 0)
	var consumerWG sync.WaitGroup
	consumerWG.Add(numOfMessagesToSend)
	processorFunc := func(batch kafka.Batch) error {
		for _, msg := range batch.Messages() {
			var msgContent string
			err := decodeString(msg.Message().Value, &msgContent)
			assert.NoError(t, err)
			actualSuccessfulMessages = append(actualSuccessfulMessages, msgContent)
			consumerWG.Done()
		}
		return nil
	}
	chNotif := make(chan struct{})
	component := newSimpleComponent(t, successTopic3, 3, 10, processorFunc, simple.NotificationOnceReachingLatestOffset(chNotif))

	// Send messages to the kafka topic
	producer, err := NewProducer()
	require.NoError(t, err)
	for i := 1; i <= numOfMessagesToSend; i++ {
		_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: successTopic3, Value: sarama.StringEncoder(strconv.Itoa(i))})
		require.NoError(t, err)
	}

	// Run Patron with the kafka component
	patronContext, patronCancel := context.WithCancel(context.Background())
	var patronWG sync.WaitGroup
	patronWG.Add(1)
	go func() {
		svc, err := patron.New(successTopic3, "0", patron.LogFields(map[string]interface{}{"test": successTopic3}))
		require.NoError(t, err)
		err = svc.WithComponents(component).Run(patronContext)
		require.NoError(t, err)
		patronWG.Done()
	}()

	// Wait for consumer to finish processing all the messages.
	consumerWG.Wait()

	// Verify all messages were processed in the right order
	expectedMessages := make([]string, numOfMessagesToSend)
	for i := 0; i < numOfMessagesToSend; i++ {
		expectedMessages[i] = strconv.Itoa(i + 1)
	}
	assert.Equal(t, expectedMessages, actualSuccessfulMessages)
	// At this stage, we have received all the expected messages.
	// We should also check that the notification channel is also eventually closed.
	select {
	case <-time.After(time.Second):
		assert.FailNow(t, "notification channel not closed")
	case _, open := <-chNotif:
		assert.False(t, open)
	}

	// Shutdown Patron and wait for it to finish
	patronCancel()
	patronWG.Wait()
}

func newSimpleComponent(t *testing.T, name string, retries uint, batchSize uint, processorFunc kafka.BatchProcessorFunc, oo ...simple.OptionFunc) *simple.Component {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaCfg.Version = sarama.V2_6_0_0

	broker := fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)
	options := []simple.OptionFunc{
		simple.FailureStrategy(kafka.ExitStrategy),
		simple.BatchSize(batchSize),
		simple.BatchTimeout(100 * time.Millisecond),
		simple.Retries(retries),
		simple.RetryWait(200 * time.Millisecond),
		simple.SaramaConfig(saramaCfg),
	}
	options = append(options, oo...)
	cmp, err := simple.New(
		name,
		[]string{broker},
		name,
		processorFunc,
		options...,
	)
	require.NoError(t, err)

	return cmp
}
