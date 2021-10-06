//go:build integration
// +build integration

package kafka

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron"
	v2 "github.com/beatlabs/patron/client/kafka/v2"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/component/kafka/group"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKafkaComponent_Success(t *testing.T) {
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
	component := newComponent(t, successTopic2, 3, 10, processorFunc)

	// Run Patron with the kafka component
	patronContext, patronCancel := context.WithCancel(context.Background())
	var patronWG sync.WaitGroup
	patronWG.Add(1)
	go func() {
		svc, err := patron.New(successTopic2, "0", patron.LogFields(map[string]interface{}{"test": successTopic2}))
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
			_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: successTopic2, Value: sarama.StringEncoder(strconv.Itoa(i))})
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

func TestKafkaComponent_FailAllRetries(t *testing.T) {
	// Test parameters
	numOfMessagesToSend := 100
	errAtIndex := 70

	// Set up the kafka component
	actualSuccessfulMessages := make([]int, 0)
	actualNumOfRuns := int32(0)
	processorFunc := func(batch kafka.Batch) error {
		for _, msg := range batch.Messages() {
			var msgContent string
			err := decodeString(msg.Message().Value, &msgContent)
			assert.NoError(t, err)

			msgIndex, err := strconv.Atoi(msgContent)
			assert.NoError(t, err)

			if msgIndex == errAtIndex {
				atomic.AddInt32(&actualNumOfRuns, 1)
				return errors.New("expected error")
			}
			actualSuccessfulMessages = append(actualSuccessfulMessages, msgIndex)
		}
		return nil
	}

	numOfRetries := uint(3)
	batchSize := uint(1)
	component := newComponent(t, failAllRetriesTopic2, numOfRetries, batchSize, processorFunc)

	// Send messages to the kafka topic
	var producerWG sync.WaitGroup
	producerWG.Add(1)
	go func() {
		producer, err := NewProducer()
		require.NoError(t, err)
		for i := 1; i <= numOfMessagesToSend; i++ {
			_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: failAllRetriesTopic2, Value: sarama.StringEncoder(strconv.Itoa(i))})
			require.NoError(t, err)
		}
		producerWG.Done()
	}()

	// Run Patron with the component - no need for goroutine since we expect it to stop after the retries fail
	svc, err := patron.New(failAllRetriesTopic2, "0", patron.LogFields(map[string]interface{}{"test": failAllRetriesTopic2}))
	require.NoError(t, err)
	err = svc.WithComponents(component).Run(context.Background())
	assert.Error(t, err)

	// Wait for the producer & consumer to finish
	producerWG.Wait()

	// Verify all messages were processed in the right order
	expectedMessages := make([]int, errAtIndex-1)
	for i := 0; i < errAtIndex-1; i++ {
		expectedMessages[i] = i + 1
	}
	assert.Equal(t, expectedMessages, actualSuccessfulMessages)
	assert.Equal(t, int32(numOfRetries+1), actualNumOfRuns)
}

func TestKafkaComponent_FailOnceAndRetry(t *testing.T) {
	// Test parameters
	numOfMessagesToSend := 100

	// Set up the component
	didFail := int32(0)
	actualMessages := make([]string, 0)
	var consumerWG sync.WaitGroup
	consumerWG.Add(numOfMessagesToSend)
	processorFunc := func(batch kafka.Batch) error {
		for _, msg := range batch.Messages() {
			var msgContent string
			err := decodeString(msg.Message().Value, &msgContent)
			assert.NoError(t, err)

			if msgContent == "50" && atomic.CompareAndSwapInt32(&didFail, 0, 1) {
				return errors.New("expected error")
			}
			consumerWG.Done()
			actualMessages = append(actualMessages, msgContent)
		}
		return nil
	}
	component := newComponent(t, failAndRetryTopic2, 3, 1, processorFunc)

	// Send messages to the kafka topic
	var producerWG sync.WaitGroup
	producerWG.Add(1)
	go func() {
		producer, err := NewProducer()
		require.NoError(t, err)
		for i := 1; i <= numOfMessagesToSend; i++ {
			_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: failAndRetryTopic2, Value: sarama.StringEncoder(strconv.Itoa(i))})
			require.NoError(t, err)
		}
		producerWG.Done()
	}()

	// Run Patron with the component
	patronContext, patronCancel := context.WithCancel(context.Background())
	var patronWG sync.WaitGroup
	patronWG.Add(1)
	go func() {
		svc, err := patron.New(failAndRetryTopic2, "0", patron.LogFields(map[string]interface{}{"test": failAndRetryTopic2}))
		require.NoError(t, err)
		err = svc.WithComponents(component).Run(patronContext)
		require.NoError(t, err)
		patronWG.Done()
	}()

	// Wait for the producer & consumer to finish
	producerWG.Wait()
	consumerWG.Wait()

	// Shutdown Patron and wait for it to finish
	patronCancel()
	patronWG.Wait()

	// Verify all messages were processed in the right order
	expectedMessages := make([]string, numOfMessagesToSend)
	for i := 0; i < numOfMessagesToSend; i++ {
		expectedMessages[i] = strconv.Itoa(i + 1)
	}
	assert.Equal(t, expectedMessages, actualMessages)
}

func newComponent(t *testing.T, name string, retries uint, batchSize uint, processorFunc kafka.BatchProcessorFunc) *group.Component {
	saramaCfg, err := v2.DefaultConsumerSaramaConfig(name, true)
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaCfg.Version = sarama.V2_6_0_0
	require.NoError(t, err)

	broker := fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)
	cmp, err := group.New(
		name,
		name+"-group",
		[]string{broker},
		[]string{name},
		processorFunc,
		saramaCfg,
		group.FailureStrategy(kafka.ExitStrategy),
		group.BatchSize(batchSize),
		group.BatchTimeout(100*time.Millisecond),
		group.Retries(retries),
		group.RetryWait(200*time.Millisecond),
		group.CommitSync())
	require.NoError(t, err)

	return cmp
}

func decodeString(data []byte, v interface{}) error {
	tmp := string(data)
	p, ok := v.(*string)
	if !ok {
		return errors.New("not a string")
	}
	*p = tmp
	return nil
}
