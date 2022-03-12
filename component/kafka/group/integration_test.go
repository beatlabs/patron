//go:build integration
// +build integration

package group

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	successTopic2        = "successTopic2"
	failAllRetriesTopic2 = "failAllRetriesTopic2"
	failAndRetryTopic2   = "failAndRetryTopic2"
	successTopic3        = "successTopic3"
	broker               = "127.0.0.1:9093"
)

func TestKafkaComponent_Success(t *testing.T) {
	require.NoError(t, test.CreateTopics(broker, successTopic2))
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
		producer, err := test.NewProducer(broker)
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
	require.NoError(t, test.CreateTopics(broker, failAllRetriesTopic2))
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

	producer, err := test.NewProducer(broker)
	require.NoError(t, err)

	msgs := make([]*sarama.ProducerMessage, 0, numOfMessagesToSend)

	for i := 1; i <= numOfMessagesToSend; i++ {
		msgs = append(msgs, &sarama.ProducerMessage{Topic: failAllRetriesTopic2, Value: sarama.StringEncoder(strconv.Itoa(i))})
	}

	err = producer.SendMessages(msgs)
	require.NoError(t, err)

	// Run Patron with the component - no need for goroutine since we expect it to stop after the retries fail
	svc, err := patron.New(failAllRetriesTopic2, "0", patron.LogFields(map[string]interface{}{"test": failAllRetriesTopic2}))
	require.NoError(t, err)
	err = svc.WithComponents(component).Run(context.Background())
	assert.Error(t, err)

	// Verify all messages were processed in the right order
	expectedMessages := make([]int, errAtIndex-1)
	for i := 0; i < errAtIndex-1; i++ {
		expectedMessages[i] = i + 1
	}
	assert.Equal(t, expectedMessages, actualSuccessfulMessages)
	assert.Equal(t, int32(numOfRetries+1), actualNumOfRuns)
}

func TestKafkaComponent_FailOnceAndRetry(t *testing.T) {
	require.NoError(t, test.CreateTopics(broker, failAndRetryTopic2))
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
		producer, err := test.NewProducer(broker)
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

func TestGroupConsume_CheckTopicFailsDueToNonExistingTopic(t *testing.T) {
	// Test parameters
	processorFunc := func(batch kafka.Batch) error {
		return nil
	}
	invalidTopicName := "invalid-topic-name"
	_, err := New(invalidTopicName, invalidTopicName+"-group", []string{broker},
		[]string{invalidTopicName}, processorFunc, sarama.NewConfig(), CheckTopic())
	require.EqualError(t, err, "topic invalid-topic-name does not exist in broker")
}

func TestGroupConsume_CheckTopicFailsDueToNonExistingBroker(t *testing.T) {
	// Test parameters
	processorFunc := func(batch kafka.Batch) error {
		return nil
	}
	_, err := New(successTopic3, successTopic3+"-group", []string{"127.0.0.1:9999"},
		[]string{successTopic3}, processorFunc, sarama.NewConfig(), CheckTopic())
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "failed to create client:")
}

func newComponent(t *testing.T, name string, retries uint, batchSize uint, processorFunc kafka.BatchProcessorFunc) *Component {
	saramaCfg, err := kafka.DefaultConsumerSaramaConfig(name, true)
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaCfg.Version = sarama.V2_6_0_0
	require.NoError(t, err)

	cmp, err := New(name, name+"-group", []string{broker}, []string{name}, processorFunc,
		saramaCfg, FailureStrategy(kafka.ExitStrategy), BatchSize(batchSize), BatchTimeout(100*time.Millisecond),
		Retries(retries), RetryWait(200*time.Millisecond), CommitSync(), CheckTopic())
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
