//go:build integration
// +build integration

package kafka

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/component/async/kafka"
	"github.com/beatlabs/patron/component/async/kafka/simple"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKafkaAsyncPackageSimpleComponent_Success(t *testing.T) {
	// Test parameters
	numOfMessagesToSend := 100

	// Set up the kafka component
	actualSuccessfulMessages := make([]string, 0)
	var consumerWG sync.WaitGroup
	consumerWG.Add(numOfMessagesToSend)
	processorFunc := func(msg async.Message) error {
		var msgContent string
		err := msg.Decode(&msgContent)
		assert.NoError(t, err)
		actualSuccessfulMessages = append(actualSuccessfulMessages, msgContent)
		consumerWG.Done()
		return nil
	}
	component := newKafkaAsyncPackageSimpleComponent(t, successTopic3, 3, processorFunc)

	// Run Patron with the kafka component
	patronContext, patronCancel := context.WithCancel(context.Background())
	var patronWG sync.WaitGroup
	patronWG.Add(1)
	go func() {
		svc, err := patron.New(successTopic3, "0", patron.TextLogger())
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

func newKafkaAsyncPackageSimpleComponent(t *testing.T, name string, retries uint, processorFunc func(message async.Message) error) *async.Component {
	decode := func(data []byte, v interface{}) error {
		tmp := string(data)
		p := v.(*string)
		*p = tmp
		return nil
	}
	factory, err := simple.New(
		name,
		name,
		Brokers(),
		kafka.Decoder(decode),
		kafka.Start(sarama.OffsetOldest))
	require.NoError(t, err)

	cmp, err := async.New(name, factory, processorFunc).
		WithRetries(retries).
		WithRetryWait(200 * time.Millisecond).
		WithFailureStrategy(async.NackExitStrategy).
		Create()
	require.NoError(t, err)

	return cmp
}
