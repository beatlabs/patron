// Package simple provides a simple consumer implementation without consumer groups.
package simple

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/beatlabs/patron/log"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/kafka"
	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/internal/validation"
)

const (
	consumerComponent = "kafka-consumer"
	subsystem         = "kafka-simple"
	messageReceived   = "received"
	messageProcessed  = "processed"
	messageErrored    = "errored"
	messageSkipped    = "skipped"
)

const (
	defaultRetries         = 3
	defaultRetryWait       = 10 * time.Second
	defaultBatchSize       = 1
	defaultBatchTimeout    = 100 * time.Millisecond
	defaultFailureStrategy = kafka.ExitStrategy
)

var (
	consumerErrors           *prometheus.CounterVec
	topicPartitionOffsetDiff *prometheus.GaugeVec
	messageStatus            *prometheus.CounterVec
)

func init() {
	consumerErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: subsystem,
			Name:      "consumer_errors",
			Help:      "Consumer errors, classified by consumer name",
		},
		[]string{"name"},
	)

	topicPartitionOffsetDiff = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "component",
			Subsystem: subsystem,
			Name:      "offset_diff",
			Help:      "Message offset difference with high watermark, classified by topic and partition",
		},
		[]string{"group", "topic", "partition"},
	)

	messageStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "component",
			Subsystem: subsystem,
			Name:      "message_status",
			Help:      "Message status counter (received, processed, errored) classified by topic and partition",
		}, []string{"status", "partition", "topic"},
	)

	prometheus.MustRegister(consumerErrors, topicPartitionOffsetDiff, messageStatus)
}

type Component struct {
	name         string
	topic        string
	brokers      []string
	saramaConfig *sarama.Config
	proc         kafka.BatchProcessorFunc
	failStrategy kafka.FailStrategy
	batchSize    uint
	batchTimeout time.Duration
	retries      uint
	retryWait    time.Duration
	// WithDurationOffset options
	durationBasedConsumer bool
	durationOffset        time.Duration
	timeExtractor         func(*sarama.ConsumerMessage) (time.Time, error)
	// WithNotificationOnceReachingLatestOffset options
	latestOffsetReachedChan chan<- struct{}
	latestOffsets           map[int32]int64
	startingOffsets         map[int32]int64
}

// New initializes a new simple kafka consumer component with support for functional configuration.
// The default failure strategy is the ExitStrategy.
// The default batch size is 1 and the batch timeout is 100ms.
// The default number of retries is 0 and the retry wait is 0.
func New(name string, brokers []string, topic string, proc kafka.BatchProcessorFunc, oo ...OptionFunc) (*Component, error) {
	var errs []error
	if name == "" {
		errs = append(errs, errors.New("name is required"))
	}

	if validation.IsStringSliceEmpty(brokers) {
		errs = append(errs, errors.New("brokers are empty or have an empty value"))
	}

	if topic == "" {
		errs = append(errs, errors.New("topic is required"))
	}

	if proc == nil {
		errs = append(errs, errors.New("work processor is required"))
	}

	if len(errs) > 0 {
		return nil, patronErrors.Aggregate(errs...)
	}

	defaultSaramaCfg, err := kafka.DefaultSaramaConfig(name)
	if err != nil {
		return nil, err
	}

	cmp := &Component{
		name:         name,
		brokers:      brokers,
		topic:        topic,
		proc:         proc,
		retries:      defaultRetries,
		retryWait:    defaultRetryWait,
		batchSize:    defaultBatchSize,
		batchTimeout: defaultBatchTimeout,
		failStrategy: defaultFailureStrategy,
		saramaConfig: defaultSaramaCfg,
	}

	for _, optionFunc := range oo {
		err = optionFunc(cmp)
		if err != nil {
			return nil, err
		}
	}

	return cmp, nil
}

// Run starts the consumer processing loop to process messages from Kafka.
func (c *Component) Run(ctx context.Context) error {
	retries := int(c.retries)

	if retries == 0 {
		return c.runWithoutRetry(ctx)
	}
	return c.runWithRetry(ctx, retries)
}

func (c *Component) runWithoutRetry(ctx context.Context) error {
	_, err := c.processing(ctx)
	if err != nil {
		return fmt.Errorf("simple consumer error on topic %s: %w", c.topic, err)
	}
	return nil
}

func (c *Component) runWithRetry(ctx context.Context, retries int) error {
	var hasProcessedMessages bool
	var err error

	for i := 0; i <= retries; i++ {
		hasProcessedMessages, err = c.processing(ctx)
		if err != nil {
			log.Errorf("simple consumer error on topic %s: %v", c.topic, err)
		}

		if hasProcessedMessages {
			i = 0
		}
		log.Errorf("failed run, retry %d/%d with %v wait", i, c.retries, c.retryWait)
		time.Sleep(c.retryWait)
	}

	return err
}

func (c *Component) processing(ctx context.Context) (hasProcessedMessages bool, err error) {
	client, consumer, pcs, err := c.createPartitionConsumers(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to create partition consumers: %w", err)
	}

	defer closeClient(client)
	defer closeConsumer(consumer)
	defer closePartitionConsumers(pcs)

	handler := newConsumerHandler(c, pcs)
	return handler.consume(ctx)
}

func (c *Component) createPartitionConsumers(ctx context.Context) (sarama.Client, sarama.Consumer, map[int32]sarama.PartitionConsumer, error) {
	client, consumer, err := c.newConsumer()
	if err != nil {
		return nil, nil, nil, err
	}

	pcs, err := c.consumePartitions(ctx, client, consumer)
	if err != nil {
		closeClient(client)
		closeConsumer(consumer)

		return nil, nil, nil, fmt.Errorf("failed to consume partitions: %w", err)
	}

	return client, consumer, pcs, nil
}

func closeClient(client sarama.Client) {
	if err := client.Close(); err != nil {
		log.Errorf("failed to close client: %v", err)
	}
}

func closeConsumer(consumer sarama.Consumer) {
	if err := consumer.Close(); err != nil {
		log.Errorf("failed to close consumer: %v", err)
	}
}

func closePartitionConsumers(pcs map[int32]sarama.PartitionConsumer) {
	for partition, pc := range pcs {
		if err := pc.Close(); err != nil {
			log.Errorf("failed to close partition consumer %d: %v", partition, err)
		}
	}
}

func (c *Component) consumePartitions(ctx context.Context, client sarama.Client, consumer sarama.Consumer) (map[int32]sarama.PartitionConsumer, error) {
	partitions, err := consumer.Partitions(c.topic)
	if err != nil {
		return nil, fmt.Errorf("failed to get partitions: %w", err)
	}

	// When kafka cluster is not fully initialized, we may get 0 partitions.
	if len(partitions) == 0 {
		return nil, errors.New("got zero partitions")
	}

	offsets, err := c.getStartingOffsets(ctx, client, consumer, partitions)
	if err != nil {
		return nil, fmt.Errorf("failed to get starting offsets: %w", err)
	}

	pcs := make(map[int32]sarama.PartitionConsumer, len(partitions))
	for _, partition := range partitions {
		pc, err := consumer.ConsumePartition(c.topic, partition, offsets[partition])
		if nil != err {
			return nil, fmt.Errorf("failed to get partition consumer: %w", err)
		}
		pcs[partition] = pc
	}

	if c.notificationOnceReachingLatestOffset() {
		c.startingOffsets = offsets
		err := c.setLatestOffsets(client, partitions)
		if err != nil {
			return nil, fmt.Errorf("failed to set latest offsets: %w", err)
		}
	}

	return pcs, nil
}

func (c *Component) getStartingOffsets(ctx context.Context, client sarama.Client, consumer sarama.Consumer, partitions []int32) (map[int32]int64, error) {
	if c.durationBasedConsumer {
		return c.getTimeBasedOffsetsPerPartition(ctx, client, consumer, partitions)
	}
	return c.getOffsetsFromInitialOffset(client, partitions)
}

func (c *Component) getOffsetsFromInitialOffset(client sarama.Client, partitions []int32) (map[int32]int64, error) {
	offsets := make(map[int32]int64)
	for _, partitionID := range partitions {
		offset, err := client.GetOffset(c.topic, partitionID, c.saramaConfig.Consumer.Offsets.Initial)
		if err != nil {
			return nil, fmt.Errorf("failed to get offset for partition %d: %w", partitionID, err)
		}
		offsets[partitionID] = offset
	}

	return offsets, nil
}

func (c *Component) getTimeBasedOffsetsPerPartition(ctx context.Context, client sarama.Client, consumer sarama.Consumer, partitions []int32) (map[int32]int64, error) {
	dkc, err := newDurationKafkaClient(client, consumer, c.saramaConfig.Net.DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka duration client: %w", err)
	}

	dc, err := newDurationClient(dkc, partitions)
	if err != nil {
		return nil, fmt.Errorf("failed to create duration client: %w", err)
	}

	offsets, err := dc.getTimeBasedOffsetsPerPartition(ctx, c.topic, time.Now().Add(-c.durationOffset), c.timeExtractor)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve duration offsets per partition: %w", err)
	}

	return offsets, nil
}

func (c *Component) newConsumer() (sarama.Client, sarama.Consumer, error) {
	client, err := sarama.NewClient(c.brokers, c.saramaConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client: %w", err)
	}

	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create simple consumer: %w", err)
	}

	return client, consumer, nil
}

func (c *Component) setLatestOffsets(client sarama.Client, partitions []int32) error {
	offsets := make(map[int32]int64)
	for _, partitionID := range partitions {
		offset, err := client.GetOffset(c.topic, partitionID, sarama.OffsetNewest)
		if err != nil {
			return err
		}
		// At this stage, offset is the offset of the next message in the partition
		offsets[partitionID] = offset - 1
	}

	c.latestOffsets = offsets
	return nil
}

func (c *Component) notificationOnceReachingLatestOffset() bool {
	return c.latestOffsetReachedChan != nil
}

// messageStatusCountInc increments the messageStatus counter for a certain status.
func messageStatusCountInc(status string, partition int32, topic string) {
	messageStatus.WithLabelValues(status, strconv.Itoa(int(partition)), topic).Inc()
}
