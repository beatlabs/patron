//go:build integration
// +build integration

package kafka

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IBM/sarama"
	kafkaclient "github.com/beatlabs/patron/client/kafka"
	"github.com/beatlabs/patron/correlation"
	testkafka "github.com/beatlabs/patron/test/kafka"

	// "github.com/opentracing/opentracing-go"

	// "github.com/opentracing/opentracing-go/mocktracer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	successTopic1        = "successTopic1"
	successTopic2        = "successTopic2"
	failAllRetriesTopic2 = "failAllRetriesTopic2"
	failAndRetryTopic2   = "failAndRetryTopic2"
	broker               = "127.0.0.1:9093"
	groupSuffix          = "-group"
)

func TestKafkaComponent_Success(t *testing.T) {
	require.NoError(t, testkafka.CreateTopics(broker, successTopic1))
	// mtr := mocktracer.New()
	// opentracing.SetGlobalTracer(mtr)
	// mtr.Reset()
	// t.Cleanup(func() { mtr.Reset() })

	// Test parameters
	numOfMessagesToSend := 100
	ctx := correlation.ContextWithID(context.Background(), "123")

	messages := make([]*sarama.ProducerMessage, 0, numOfMessagesToSend)
	for i := 1; i <= numOfMessagesToSend; i++ {
		messages = append(messages, &sarama.ProducerMessage{
			Topic:   successTopic1,
			Value:   sarama.StringEncoder(strconv.Itoa(i)),
			Headers: make([]sarama.RecordHeader, 0),
		})
	}
	cfg, err := kafkaclient.DefaultProducerSaramaConfig("test-client", true)
	require.NoError(t, err)
	client, err := kafkaclient.New([]string{broker}, cfg).Create()
	require.NoError(t, err)
	defer client.Close()

	for _, msg := range messages {
		_, _, err := client.Send(ctx, msg)
		require.NoError(t, err)
	}

	// mtr.Reset()

	// var mu sync.Mutex

	// // Set up the kafka component
	// actualSuccessfulMessages := make([]string, 0)
	// var consumerWG sync.WaitGroup
	// consumerWG.Add(numOfMessagesToSend)
	// processorFunc := func(batch Batch) error {
	// 	for _, msg := range batch.Messages() {
	// 		var msgContent string
	// 		err := decodeString(msg.Message().Value, &msgContent)
	// 		assert.NoError(t, err)
	// 		mu.Lock()
	// 		actualSuccessfulMessages = append(actualSuccessfulMessages, msgContent)
	// 		mu.Unlock()
	// 		consumerWG.Done()
	// 	}
	// 	return nil
	// }
	// component := newComponent(t, successTopic1, 3, 10, processorFunc)

	// // Run Patron with the kafka component
	// patronContext, patronCancel := context.WithCancel(context.Background())
	// var patronWG sync.WaitGroup
	// patronWG.Add(1)
	// go func() {
	// 	err := component.Run(patronContext)
	// 	require.NoError(t, err)
	// 	patronWG.Done()
	// }()

	// // Wait for both consumer and producer to finish processing all the messages.
	// consumerWG.Wait()

	// // Verify all messages were processed in the right order
	// expectedMessages := make([]string, numOfMessagesToSend)
	// for i := 0; i < numOfMessagesToSend; i++ {
	// 	expectedMessages[i] = strconv.Itoa(i + 1)
	// }
	// assert.Equal(t, expectedMessages, actualSuccessfulMessages)

	// // Shutdown Patron and wait for it to finish
	// patronCancel()
	// patronWG.Wait()

	// assert.Len(t, mtr.FinishedSpans(), 100)

	// expectedTags := map[string]interface{}{
	// 	"component":     "kafka-consumer",
	// 	"correlationID": "123",
	// 	"error":         false,
	// 	"span.kind":     ext.SpanKindEnum("consumer"),
	// 	"version":       "dev",
	// }

	// for _, span := range mtr.FinishedSpans() {
	// 	assert.Equal(t, expectedTags, span.Tags())
	// }

	// assert.GreaterOrEqual(t, testutil.CollectAndCount(consumerErrors, "component_kafka_consumer_errors"), 0)
	// assert.GreaterOrEqual(t, testutil.CollectAndCount(topicPartitionOffsetDiff, "component_kafka_offset_diff"), 1)
	// assert.GreaterOrEqual(t, testutil.CollectAndCount(messageStatus, "component_kafka_message_status"), 1)
}

func TestKafkaComponent_FailAllRetries(t *testing.T) {
	require.NoError(t, testkafka.CreateTopics(broker, failAllRetriesTopic2))
	// Test parameters
	numOfMessagesToSend := 100
	errAtIndex := 70

	// Set up the kafka component
	actualSuccessfulMessages := make([]int, 0)
	actualNumOfRuns := int32(0)
	processorFunc := func(batch Batch) error {
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

	producer, err := testkafka.NewProducer(broker)
	require.NoError(t, err)

	msgs := make([]*sarama.ProducerMessage, 0, numOfMessagesToSend)

	for i := 1; i <= numOfMessagesToSend; i++ {
		msgs = append(msgs, &sarama.ProducerMessage{Topic: failAllRetriesTopic2, Value: sarama.StringEncoder(strconv.Itoa(i))})
	}

	err = producer.SendMessages(msgs)
	require.NoError(t, err)

	err = component.Run(context.Background())
	assert.Error(t, err)

	// Verify all messages were processed in the right order
	for i := 0; i < len(actualSuccessfulMessages)-1; i++ {
		if actualSuccessfulMessages[i+1] > errAtIndex {
			assert.Fail(t, "message higher than expected", "i is %d and i+1 is %d", actualSuccessfulMessages[i+1],
				errAtIndex)
		}

		diff := actualSuccessfulMessages[i+1] - actualSuccessfulMessages[i]
		if diff == 0 || diff == 1 {
			continue
		}
		assert.Fail(t, "messages order is not correct", "i is %d and i+1 is %d", actualSuccessfulMessages[i],
			actualSuccessfulMessages[i+1])
	}

	assert.Equal(t, int32(numOfRetries+1), actualNumOfRuns)
}

func TestKafkaComponent_FailOnceAndRetry(t *testing.T) {
	require.NoError(t, testkafka.CreateTopics(broker, failAndRetryTopic2))
	// Test parameters
	numOfMessagesToSend := 100

	// Set up the component
	didFail := int32(0)
	actualMessages := make([]int, 0)
	var consumerWG sync.WaitGroup
	consumerWG.Add(numOfMessagesToSend)
	processorFunc := func(batch Batch) error {
		for _, msg := range batch.Messages() {
			var msgContent string
			err := decodeString(msg.Message().Value, &msgContent)
			assert.NoError(t, err)

			msgIndex, err := strconv.Atoi(msgContent)
			assert.NoError(t, err)

			if msgIndex == 50 && atomic.CompareAndSwapInt32(&didFail, 0, 1) {
				return errors.New("expected error")
			}
			consumerWG.Done()
			actualMessages = append(actualMessages, msgIndex)
		}
		return nil
	}
	component := newComponent(t, failAndRetryTopic2, 3, 1, processorFunc)

	// Send messages to the kafka topic
	var producerWG sync.WaitGroup
	producerWG.Add(1)
	go func() {
		producer, err := testkafka.NewProducer(broker)
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
		require.NoError(t, component.Run(patronContext))
		patronWG.Done()
	}()

	// Wait for the producer & consumer to finish
	producerWG.Wait()
	consumerWG.Wait()

	// Shutdown Patron and wait for it to finish
	patronCancel()
	patronWG.Wait()

	// Verify all messages were processed in the right order
	for i := 0; i < len(actualMessages)-1; i++ {
		diff := actualMessages[i+1] - actualMessages[i]
		if diff == 0 || diff == 1 {
			continue
		}
		assert.Fail(t, "messages order is not correct", "i is %d and i+1 is %d", actualMessages[i], actualMessages[i+1])
	}
}

func TestGroupConsume_CheckTopicFailsDueToNonExistingTopic(t *testing.T) {
	// Test parameters
	processorFunc := func(batch Batch) error {
		return nil
	}
	invalidTopicName := "invalid-topic-name"
	_, err := New(invalidTopicName, invalidTopicName+groupSuffix, []string{broker},
		[]string{invalidTopicName}, processorFunc, sarama.NewConfig(), WithCheckTopic())
	require.EqualError(t, err, "topic invalid-topic-name does not exist in broker")
}

func TestGroupConsume_CheckTopicFailsDueToNonExistingBroker(t *testing.T) {
	// Test parameters
	processorFunc := func(batch Batch) error {
		return nil
	}
	_, err := New(successTopic2, successTopic2+groupSuffix, []string{"127.0.0.1:9999"},
		[]string{successTopic2}, processorFunc, sarama.NewConfig(), WithCheckTopic())
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "failed to create client:")
}

func newComponent(t *testing.T, name string, retries uint, batchSize uint, processorFunc BatchProcessorFunc) *Component {
	saramaCfg, err := DefaultConsumerSaramaConfig(name, true)
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaCfg.Version = sarama.V2_6_0_0
	require.NoError(t, err)

	cmp, err := New(name, name+groupSuffix, []string{broker}, []string{name}, processorFunc,
		saramaCfg, WithFailureStrategy(ExitStrategy), WithBatchSize(batchSize), WithBatchTimeout(100*time.Millisecond),
		WithRetries(retries), WithRetryWait(200*time.Millisecond), WithCommitSync(), WithCheckTopic())
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
