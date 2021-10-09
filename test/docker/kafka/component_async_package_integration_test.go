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
	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/component/async/kafka"
	"github.com/beatlabs/patron/component/async/kafka/group"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKafkaAsyncPackageComponent_Success(t *testing.T) {
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
	component := newKafkaAsyncPackageComponent(t, successTopic1, 3, processorFunc)

	// Run Patron with the kafka component
	patronContext, patronCancel := context.WithCancel(context.Background())
	var patronWG sync.WaitGroup
	patronWG.Add(1)
	go func() {
		svc, err := patron.New(successTopic1, "0", patron.TextLogger())
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
			_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: successTopic1, Value: sarama.StringEncoder(strconv.Itoa(i))})
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

func TestKafkaAsyncPackageComponent_FailAllRetries(t *testing.T) {
	// Test parameters
	numOfMessagesToSend := 100
	errAtIndex := 50

	// Set up the kafka component
	actualSuccessfulMessages := make([]int, 0)
	actualNumOfRuns := int32(0)
	processorFunc := func(msg async.Message) error {
		var msgContentStr string
		err := msg.Decode(&msgContentStr)
		assert.NoError(t, err)

		msgIndex, err := strconv.Atoi(msgContentStr)
		assert.NoError(t, err)

		if msgIndex == errAtIndex {
			atomic.AddInt32(&actualNumOfRuns, 1)
			return errors.New("expected error")
		}
		actualSuccessfulMessages = append(actualSuccessfulMessages, msgIndex)
		return nil
	}
	numOfRetries := uint(3)
	component := newKafkaAsyncPackageComponent(t, failAllRetriesTopic1, numOfRetries, processorFunc)

	// Send messages to the kafka topic
	var producerWG sync.WaitGroup
	producerWG.Add(1)
	go func() {
		producer, err := NewProducer()
		require.NoError(t, err)
		for i := 1; i <= numOfMessagesToSend; i++ {
			_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: failAllRetriesTopic1, Value: sarama.StringEncoder(strconv.Itoa(i))})
			require.NoError(t, err)
		}
		producerWG.Done()
	}()

	// Run Patron with the component - no need for goroutine since we expect it to stop after the retries fail
	svc, err := patron.New(failAllRetriesTopic1, "0", patron.TextLogger())
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

func TestKafkaAsyncPackageComponent_FailOnceAndRetry(t *testing.T) {
	// Test parameters
	numOfMessagesToSend := 100

	// Set up the component
	didFail := int32(0)
	actualMessages := make([]string, 0)
	var consumerWG sync.WaitGroup
	consumerWG.Add(numOfMessagesToSend)
	processorFunc := func(msg async.Message) error {
		var msgContent string
		err := msg.Decode(&msgContent)
		assert.NoError(t, err)

		if msgContent == "50" && atomic.CompareAndSwapInt32(&didFail, 0, 1) {
			return errors.New("expected error")
		}
		consumerWG.Done()
		actualMessages = append(actualMessages, msgContent)
		return nil
	}
	component := newKafkaAsyncPackageComponent(t, failAndRetryTopic1, 3, processorFunc)

	// Send messages to the kafka topic
	var producerWG sync.WaitGroup
	producerWG.Add(1)
	go func() {
		producer, err := NewProducer()
		require.NoError(t, err)
		for i := 1; i <= numOfMessagesToSend; i++ {
			_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: failAndRetryTopic1, Value: sarama.StringEncoder(strconv.Itoa(i))})
			require.NoError(t, err)
		}
		producerWG.Done()
	}()

	// Run Patron with the component
	patronContext, patronCancel := context.WithCancel(context.Background())
	var patronWG sync.WaitGroup
	patronWG.Add(1)
	go func() {
		svc, err := patron.New(failAndRetryTopic1, "0", patron.TextLogger())
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

func newKafkaAsyncPackageComponent(t *testing.T, name string, retries uint, processorFunc func(message async.Message) error) *async.Component {
	decode := func(data []byte, v interface{}) error {
		tmp := string(data)
		p := v.(*string)
		*p = tmp
		return nil
	}
	factory, err := group.New(
		name,
		name+"-group",
		[]string{name},
		[]string{fmt.Sprintf("%s:%s", kafkaHost, kafkaPort)},
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
