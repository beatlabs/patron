//go:build integration
// +build integration

package group

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
	kafkacmp "github.com/beatlabs/patron/component/kafka"
	testkafka "github.com/beatlabs/patron/test/kafka"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	groupTopic1          = "groupTopic1"
	groupTopic2          = "groupTopic2"
	successTopic1        = "successTopic1"
	failAllRetriesTopic1 = "failAllRetriesTopic1"
	failAndRetryTopic1   = "failAndRetryTopic1"
	broker               = "127.0.0.1:9093"
)

func TestGroupConsume(t *testing.T) {
	require.NoError(t, testkafka.CreateTopics(broker, groupTopic1))

	sent := []string{"one", "two", "three"}
	chMessages := make(chan []string)
	chErr := make(chan error)
	go func() {
		saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig("test-group-consumer", true)
		require.NoError(t, err)

		factory, err := New("test1", uuid.New().String(), []string{groupTopic1}, []string{broker}, saramaCfg, kafka.WithDecoderJSON(),
			kafka.WithVersion(sarama.V2_1_0_0.String()), kafka.WithStartFromNewest())
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
		messages = append(messages, testkafka.CreateProducerMessage(groupTopic1, val))
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

func TestGroupConsume_ClaimMessageError(t *testing.T) {
	require.NoError(t, testkafka.CreateTopics(broker, groupTopic2))

	chMessages := make(chan []string)
	chErr := make(chan error)

	saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig("test-consumer", true)
	require.NoError(t, err)

	// Consumer will error out in ClaimMessage as no DecoderFunc has been set
	factory, err := New("test1", uuid.New().String(), []string{groupTopic2}, []string{broker}, saramaCfg,
		kafka.WithVersion(sarama.V2_1_0_0.String()), kafka.WithStartFromNewest())
	require.NoError(t, err)
	consumer, err := factory.Create()
	require.NoError(t, err)
	defer func() { _ = consumer.Close() }()

	go func() {
		received, err := testkafka.AsyncConsumeMessages(consumer, 1)
		if err != nil {
			chErr <- err
			return
		}

		chMessages <- received
	}()

	time.Sleep(5 * time.Second)

	err = testkafka.SendMessages(broker, testkafka.CreateProducerMessage(groupTopic2, "321"))
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

func TestKafkaAsyncPackageComponent_Success(t *testing.T) {
	require.NoError(t, testkafka.CreateTopics(broker, successTopic1))
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
	svc, err := patron.New(successTopic1, "0", patron.WithTextLogger(), patron.WithComponents(component))
	require.NoError(t, err)

	go func() {
		err = svc.Run(patronContext)
		require.NoError(t, err)
		patronWG.Done()
	}()

	// Send messages to the kafka topic
	var producerWG sync.WaitGroup
	producerWG.Add(1)
	go func() {
		producer, err := testkafka.NewProducer(broker)
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
	require.NoError(t, testkafka.CreateTopics(broker, failAllRetriesTopic1))
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
		producer, err := testkafka.NewProducer(broker)
		require.NoError(t, err)
		for i := 1; i <= numOfMessagesToSend; i++ {
			_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: failAllRetriesTopic1, Value: sarama.StringEncoder(strconv.Itoa(i))})
			require.NoError(t, err)
		}
		producerWG.Done()
	}()

	// Run Patron with the component - no need for goroutine since we expect it to stop after the retries fail
	svc, err := patron.New(failAllRetriesTopic1, "0", patron.WithTextLogger(), patron.WithComponents(component))
	require.NoError(t, err)
	err = svc.Run(context.Background())
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
	require.NoError(t, testkafka.CreateTopics(broker, failAndRetryTopic1))
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
		producer, err := testkafka.NewProducer(broker)
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
		svc, err := patron.New(failAndRetryTopic1, "0", patron.WithTextLogger(), patron.WithComponents(component))
		require.NoError(t, err)
		err = svc.Run(patronContext)
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
		p, ok := v.(*string)
		if !ok {
			return fmt.Errorf("failed to type assert to *string %v", v)
		}
		*p = tmp
		return nil
	}

	saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig(name, true)
	require.NoError(t, err)

	factory, err := New(name, name+"-group", []string{name},
		[]string{broker}, saramaCfg, kafka.WithDecoder(decode), kafka.WithStart(sarama.OffsetOldest))
	require.NoError(t, err)

	cmp, err := async.New(name, factory, processorFunc).WithRetries(retries).
		WithRetryWait(200 * time.Millisecond).WithFailureStrategy(async.NackExitStrategy).Create()
	require.NoError(t, err)

	return cmp
}
