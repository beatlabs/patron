//go:build integration

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

	"github.com/IBM/sarama"
	kafkaclient "github.com/beatlabs/patron/client/kafka"
	"github.com/beatlabs/patron/correlation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

const (
	successTopic1        = "successTopic1"
	successTopic2        = "successTopic2"
	failAllRetriesTopic2 = "failAllRetriesTopic2"
	failAndRetryTopic2   = "failAndRetryTopic2"
	broker               = "127.0.0.1:9092"
	groupSuffix          = "-group"
)

func TestKafkaComponent_Success(t *testing.T) {
	require.NoError(t, createTopics(broker, successTopic1))

	// Setup tracing
	t.Cleanup(func() { traceExporter.Reset() })

	// Setup metrics
	// Setup metrics
	read := metricsdk.NewManualReader()
	provider := metricsdk.NewMeterProvider(metricsdk.WithReader(read))
	defer func() { require.NoError(t, provider.Shutdown(context.Background())) }()
	otel.SetMeterProvider(provider)

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
	require.NoError(t, client.SendBatch(ctx, messages))

	require.NoError(t, tracePublisher.ForceFlush(context.Background()))
	traceExporter.Reset()

	// Set up the kafka component
	actualSuccessfulMessages := make([]string, 0)
	var consumerWG sync.WaitGroup
	consumerWG.Add(numOfMessagesToSend)
	processorFunc := func(batch Batch) error {
		for _, msg := range batch.Messages() {
			var msgContent string
			err := decodeString(msg.Message().Value, &msgContent)
			require.NoError(t, err)
			actualSuccessfulMessages = append(actualSuccessfulMessages, msgContent)
			consumerWG.Done()
		}
		return nil
	}
	component := newComponent(t, successTopic1, 3, 10, processorFunc)

	// Run Patron with the kafka component
	patronContext, patronCancel := context.WithCancel(context.Background())
	var patronWG sync.WaitGroup
	patronWG.Add(1)
	go func() {
		err := component.Run(patronContext)
		assert.NoError(t, err)
		patronWG.Done()
	}()

	// Wait for both consumer and producer to finish processing all the messages.
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

	time.Sleep(time.Second)

	require.NoError(t, tracePublisher.ForceFlush(context.Background()))

	spans := traceExporter.GetSpans()

	assert.Len(t, spans, 100)

	for _, span := range spans {
		expectedSpan := tracetest.SpanStub{
			Name:     "kafka-consumer successTopic1",
			SpanKind: trace.SpanKindConsumer,
			Status: tracesdk.Status{
				Code: codes.Ok,
			},
		}

		assertSpan(t, expectedSpan, span)
	}

	// Metrics
	collectedMetrics := &metricdata.ResourceMetrics{}
	require.NoError(t, read.Collect(context.Background(), collectedMetrics))
	assert.Len(t, collectedMetrics.ScopeMetrics, 1)
	assert.Len(t, collectedMetrics.ScopeMetrics[0].Metrics, 3)
	assert.Equal(t, "kafka.publish.count", collectedMetrics.ScopeMetrics[0].Metrics[0].Name)
	assert.Equal(t, "kafka.consumer.offset.diff", collectedMetrics.ScopeMetrics[0].Metrics[1].Name)
	assert.Equal(t, "kafka.message.status", collectedMetrics.ScopeMetrics[0].Metrics[2].Name)
}

func assertSpan(t *testing.T, expected tracetest.SpanStub, got tracetest.SpanStub) {
	assert.Equal(t, expected.Name, got.Name)
	assert.Equal(t, expected.SpanKind, got.SpanKind)
	assert.Equal(t, expected.Status, got.Status)
}

func TestKafkaComponent_FailAllRetries(t *testing.T) {
	require.NoError(t, createTopics(broker, failAllRetriesTopic2))
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
			require.NoError(t, err)

			msgIndex, err := strconv.Atoi(msgContent)
			require.NoError(t, err)

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

	producer, err := newProducer(broker)
	require.NoError(t, err)

	msgs := make([]*sarama.ProducerMessage, 0, numOfMessagesToSend)

	for i := 1; i <= numOfMessagesToSend; i++ {
		msgs = append(msgs, &sarama.ProducerMessage{Topic: failAllRetriesTopic2, Value: sarama.StringEncoder(strconv.Itoa(i))})
	}

	err = producer.SendMessages(msgs)
	require.NoError(t, err)

	err = component.Run(context.Background())
	require.Error(t, err)

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
	require.NoError(t, createTopics(broker, failAndRetryTopic2))
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
			require.NoError(t, err)

			msgIndex, err := strconv.Atoi(msgContent)
			require.NoError(t, err)

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
		producer, err := newProducer(broker)
		assert.NoError(t, err)

		for i := 1; i <= numOfMessagesToSend; i++ {
			_, _, err := producer.SendMessage(&sarama.ProducerMessage{Topic: failAndRetryTopic2, Value: sarama.StringEncoder(strconv.Itoa(i))})
			assert.NoError(t, err)
		}
		producerWG.Done()
	}()

	// Run Patron with the component
	patronContext, patronCancel := context.WithCancel(context.Background())
	var patronWG sync.WaitGroup
	patronWG.Add(1)
	go func() {
		assert.NoError(t, component.Run(patronContext))
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
	processorFunc := func(_ Batch) error {
		return nil
	}
	invalidTopicName := "invalid-topic-name"
	_, err := New(invalidTopicName, invalidTopicName+groupSuffix, []string{broker},
		[]string{invalidTopicName}, processorFunc, sarama.NewConfig(), WithCheckTopic())
	require.EqualError(t, err, "topic invalid-topic-name does not exist in broker")
}

func TestGroupConsume_CheckTopicFailsDueToNonExistingBroker(t *testing.T) {
	// Test parameters
	processorFunc := func(_ Batch) error {
		return nil
	}
	_, err := New(successTopic2, successTopic2+groupSuffix, []string{"127.0.0.1:9999"},
		[]string{successTopic2}, processorFunc, sarama.NewConfig(), WithCheckTopic())
	require.Error(t, err)
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

func createTopics(broker string, topics ...string) error {
	brk := sarama.NewBroker(broker)

	err := brk.Open(sarama.NewConfig())
	if err != nil {
		return err
	}

	// check if the connection was OK
	connected, err := brk.Connected()
	if err != nil {
		return err
	}
	if !connected {
		return errors.New("not connected")
	}
	deleteReq := &sarama.DeleteTopicsRequest{
		Topics:  topics,
		Timeout: time.Second * 15,
	}

	deleteResp, err := brk.DeleteTopics(deleteReq)
	if err != nil {
		return err
	}

	for k, v := range deleteResp.TopicErrorCodes {
		if v == sarama.ErrNoError || v == sarama.ErrUnknownTopicOrPartition {
			continue
		}
		fmt.Println(k)
		fmt.Println(v)
	}

	time.Sleep(100 * time.Millisecond)

	topicDetail := &sarama.TopicDetail{}
	topicDetail.NumPartitions = int32(1)
	topicDetail.ReplicationFactor = int16(1)
	topicDetail.ConfigEntries = make(map[string]*string)

	topicDetails := make(map[string]*sarama.TopicDetail, len(topics))

	for _, topic := range topics {
		topicDetails[topic] = topicDetail
	}

	request := sarama.CreateTopicsRequest{
		Timeout:      time.Second * 15,
		TopicDetails: topicDetails,
	}

	response, err := brk.CreateTopics(&request)
	if err != nil {
		return err
	}

	for _, val := range response.TopicErrors {
		if val.Err == sarama.ErrTopicAlreadyExists || val.Err == sarama.ErrNoError {
			continue
		}
		return errors.New(val.Error())
	}

	return brk.Close()
}

func newProducer(broker string) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	return sarama.NewSyncProducer([]string{broker}, config)
}
