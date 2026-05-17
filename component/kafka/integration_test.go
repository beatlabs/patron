//go:build integration

package kafka

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	kafkaclient "github.com/beatlabs/patron/client/kafka"
	"github.com/beatlabs/patron/correlation"
	"github.com/beatlabs/patron/internal/test"
	patrontrace "github.com/beatlabs/patron/observability/trace"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/goleak"
)

var (
	tracePublisher *tracesdk.TracerProvider
	traceExporter  = tracetest.NewInMemoryExporter()
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("google.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/metric.(*PeriodicReader).run"),
		goleak.IgnoreTopFunction("go.opentelemetry.io/otel/sdk/trace.(*batchSpanProcessor).processQueue"),
		goleak.IgnoreTopFunction("github.com/twmb/franz-go/pkg/kgo.(*Client).updateMetadataLoop"),
		goleak.IgnoreTopFunction("github.com/twmb/franz-go/pkg/kgo.(*Client).reapConnectionsLoop"),
	)
}

func init() {
	if err := os.Setenv("OTEL_BSP_SCHEDULE_DELAY", "100"); err != nil {
		panic(err)
	}

	tracePublisher = patrontrace.Setup("test", nil, traceExporter)
}

const (
	successTopic1        = "successTopic1"
	successTopic2        = "successTopic2"
	failAllRetriesTopic2 = "failAllRetriesTopic2"
	failAndRetryTopic2   = "failAndRetryTopic2"
	broker               = "127.0.0.1:9092"
)

func uniqueGroup(topic string) string {
	return topic + "-group-" + uuid.New().String()
}

func TestKafkaComponent_Success(t *testing.T) {
	require.NoError(t, createTopics(broker, successTopic1))

	// Setup tracing
	t.Cleanup(func() { traceExporter.Reset() })

	ctx := correlation.ContextWithID(context.Background(), "123")

	shutdownProvider, collectMetrics := test.SetupMetrics(ctx, t)
	defer shutdownProvider()

	// Test parameters
	numOfMessagesToSend := 100

	messages := make([]*kgo.Record, 0, numOfMessagesToSend)
	for i := 1; i <= numOfMessagesToSend; i++ {
		messages = append(messages, &kgo.Record{Topic: successTopic1, Value: []byte(strconv.Itoa(i))})
	}
	client, err := kafkaclient.New([]string{broker}, kgo.RequiredAcks(kgo.AllISRAcks()))
	require.NoError(t, err)
	_, err = client.Send(ctx, messages...)
	require.NoError(t, err)

	require.NoError(t, tracePublisher.ForceFlush(context.Background()))
	traceExporter.Reset()

	// Set up the kafka component
	actualSuccessfulMessages := make([]string, 0)
	var consumerWG sync.WaitGroup
	consumerWG.Add(numOfMessagesToSend)
	processorFunc := func(_ context.Context, records []*kgo.Record) error {
		for _, rec := range records {
			var msgContent string
			err := decodeString(rec.Value, &msgContent)
			require.NoError(t, err)
			actualSuccessfulMessages = append(actualSuccessfulMessages, msgContent)
			consumerWG.Done()
		}
		return nil
	}
	component := newComponent(t, successTopic1, uniqueGroup(successTopic1), 3, 10, processorFunc)

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

	allSpans := traceExporter.GetSpans()

	patronSpanName := successTopic1 + " process"
	var patronSpans []tracetest.SpanStub
	for _, span := range allSpans {
		if span.Name == patronSpanName {
			patronSpans = append(patronSpans, span)
		}
	}

	assert.Len(t, patronSpans, 100)

	for _, span := range patronSpans {
		expectedSpan := tracetest.SpanStub{
			Name:     patronSpanName,
			SpanKind: trace.SpanKindConsumer,
			Status: tracesdk.Status{
				Code: codes.Ok,
			},
		}

		test.AssertSpan(t, expectedSpan, span)
	}

	// Metrics
	collectedMetrics := collectMetrics(3)
	allMetrics := test.AllMetrics(collectedMetrics)
	test.AssertMetric(t, allMetrics, "kafka.consumer.offset.diff")
	test.AssertMetric(t, allMetrics, "kafka.publish.count")
	test.AssertMetric(t, allMetrics, "kafka.message.status")
	assertMetricAbsent(t, allMetrics, "kafka.consumer.errors")
}

func assertMetricAbsent(t *testing.T, metrics []metricdata.Metrics, expectedName string) {
	t.Helper()
	for _, metric := range metrics {
		if metric.Name == expectedName {
			assert.Fail(t, "unexpected metric found", "metric %s should not be present", expectedName)
			return
		}
	}
}

func TestKafkaComponent_FailAllRetries(t *testing.T) {
	require.NoError(t, createTopics(broker, failAllRetriesTopic2))
	// Test parameters
	numOfMessagesToSend := 10
	errAtIndex := 7

	// Set up the kafka component
	actualSuccessfulMessages := make([]int, 0)
	actualNumOfRuns := uint32(0)
	processorFunc := func(_ context.Context, records []*kgo.Record) error {
		for _, rec := range records {
			var msgContent string
			err := decodeString(rec.Value, &msgContent)
			require.NoError(t, err)

			msgIndex, err := strconv.Atoi(msgContent)
			require.NoError(t, err)

			if msgIndex == errAtIndex {
				atomic.AddUint32(&actualNumOfRuns, 1)
				return errors.New("expected error")
			}
			actualSuccessfulMessages = append(actualSuccessfulMessages, msgIndex)
		}
		return nil
	}

	numOfRetries := uint32(1)
	batchSize := uint(1)
	component := newComponent(t, failAllRetriesTopic2, uniqueGroup(failAllRetriesTopic2), numOfRetries, batchSize, processorFunc)

	producer, err := kafkaclient.New([]string{broker}, kgo.DisableIdempotentWrite())
	require.NoError(t, err)
	defer producer.Close()

	msgs := make([]*kgo.Record, 0, numOfMessagesToSend)

	for i := 1; i <= numOfMessagesToSend; i++ {
		msgs = append(msgs, &kgo.Record{Topic: failAllRetriesTopic2, Value: []byte(strconv.Itoa(i))})
	}

	_, err = producer.Send(context.Background(), msgs...)
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

	assert.Equal(t, numOfRetries+1, actualNumOfRuns)
}

func TestKafkaComponent_FailOnceAndRetry(t *testing.T) {
	require.NoError(t, createTopics(broker, failAndRetryTopic2))
	// Test parameters
	numOfMessagesToSend := 10

	// Set up the component
	didFail := int32(0)
	actualMessages := make([]int, 0)
	var consumerWG sync.WaitGroup
	consumerWG.Add(numOfMessagesToSend)
	processorFunc := func(_ context.Context, records []*kgo.Record) error {
		for _, rec := range records {
			var msgContent string
			err := decodeString(rec.Value, &msgContent)
			require.NoError(t, err)

			msgIndex, err := strconv.Atoi(msgContent)
			require.NoError(t, err)

			if msgIndex == 5 && atomic.CompareAndSwapInt32(&didFail, 0, 1) {
				return errors.New("expected error")
			}
			consumerWG.Done()
			actualMessages = append(actualMessages, msgIndex)
		}
		return nil
	}
	component := newComponent(t, failAndRetryTopic2, uniqueGroup(failAndRetryTopic2), 1, 1, processorFunc)

	// Send messages to the kafka topic
	var producerWG sync.WaitGroup
	producerWG.Add(1)
	go func() {
		producer, err := kafkaclient.New([]string{broker}, kgo.DisableIdempotentWrite())
		assert.NoError(t, err)
		defer producer.Close()

		for i := 1; i <= numOfMessagesToSend; i++ {
			_, err := producer.Send(context.Background(), &kgo.Record{Topic: failAndRetryTopic2, Value: []byte(strconv.Itoa(i))})
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
	processorFunc := func(_ context.Context, _ []*kgo.Record) error {
		return nil
	}
	invalidTopicName := "invalid-topic-name"
	_, err := New(invalidTopicName, uniqueGroup(invalidTopicName), []string{broker},
		[]string{invalidTopicName}, processorFunc, nil, WithCheckTopic())
	require.EqualError(t, err, "topic invalid-topic-name does not exist in broker")
}

func TestGroupConsume_CheckTopicFailsDueToNonExistingBroker(t *testing.T) {
	// Test parameters
	processorFunc := func(_ context.Context, _ []*kgo.Record) error {
		return nil
	}
	_, err := New(successTopic2, uniqueGroup(successTopic2), []string{"127.0.0.1:9999"},
		[]string{successTopic2}, processorFunc, nil, WithCheckTopic())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get topics from broker:")
}

func newComponent(t *testing.T, name, group string, retries uint32, batchSize uint, processorFunc ProcessorFunc) *Component {
	t.Helper()

	opts := []kgo.Opt{
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	}

	cmp, err := New(name, group, []string{broker}, []string{name}, processorFunc,
		opts, WithFailureStrategy(ExitStrategy), WithBatchSize(batchSize), WithBatchTimeout(100*time.Millisecond),
		WithRetries(retries), WithRetryWait(200*time.Millisecond), WithManualCommit(), WithCheckTopic())
	require.NoError(t, err)

	return cmp
}

func decodeString(data []byte, v any) error {
	tmp := string(data)
	p, ok := v.(*string)
	if !ok {
		return errors.New("not a string")
	}
	*p = tmp
	return nil
}

func createTopics(broker string, topics ...string) error {
	cl, err := kgo.NewClient(kgo.SeedBrokers(broker))
	if err != nil {
		return err
	}
	defer cl.Close()

	adm := kadm.NewClient(cl)

	for _, topic := range topics {
		_, err := adm.DeleteTopics(context.Background(), topic)
		if err != nil {
			fmt.Printf("Warning: failed to delete topic %s: %v\n", topic, err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	resp, err := adm.CreateTopics(context.Background(), 1, 1, nil, topics...)
	if err != nil {
		return err
	}
	for _, topic := range resp {
		if topic.Err != nil {
			return fmt.Errorf("failed to create topic %s: %w", topic.Topic, topic.Err)
		}
	}

	return nil
}
