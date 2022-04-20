// Package simple provides a simple consumer implementation without consumer groups.
package simple

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/component/async/kafka"
	"github.com/beatlabs/patron/internal/validation"
	"github.com/beatlabs/patron/log"
)

// unixNanoToTimestampDivider divides unix nano seconds to valid timestamp for kafka messages.
const unixNanoToTimestampDivider = 1000_000

// TimeExtractor defines a function extracting a time from a Kafka message.
type TimeExtractor func(*sarama.ConsumerMessage) (time.Time, error)

// WithDurationOffset allows creating a consumer from a given duration.
// It accepts a function indicating how to extract the time from a Kafka message.
func WithDurationOffset(since time.Duration, timeExtractor TimeExtractor) kafka.OptionFunc {
	return func(c *kafka.ConsumerConfig) error {
		if since < 0 {
			return errors.New("duration must be positive")
		}
		if timeExtractor == nil {
			return errors.New("empty time extractor function")
		}
		c.DurationBasedConsumer = true
		c.DurationOffset = since
		c.TimeExtractor = timeExtractor
		return nil
	}
}

// WithTimestampOffset allows creating a consumer from a given duration.
func WithTimestampOffset(since time.Duration) kafka.OptionFunc {
	return func(c *kafka.ConsumerConfig) error {
		if since < 0 {
			return errors.New("duration must be positive")
		}
		c.TimestampBasedConsumer = true
		c.TimestampOffset = time.Now().Add(-since).UnixNano() / unixNanoToTimestampDivider
		return nil
	}
}

// WithNotificationOnceReachingLatestOffset closes the input channel once all the partition consumers have reached the
// latest offset.
func WithNotificationOnceReachingLatestOffset(ch chan<- struct{}) kafka.OptionFunc {
	return func(c *kafka.ConsumerConfig) error {
		if ch == nil {
			return errors.New("nil channel")
		}
		c.LatestOffsetReachedChan = ch
		return nil
	}
}

// Factory definition of a consumer factory.
type Factory struct {
	name      string
	topic     string
	brokers   []string
	saramaCfg *sarama.Config
	oo        []kafka.OptionFunc
}

// New constructor.
func New(name, topic string, brokers []string, saramaCfg *sarama.Config, oo ...kafka.OptionFunc) (*Factory, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}

	if saramaCfg == nil {
		return nil, errors.New("no Sarama configuration specified")
	}

	if validation.IsStringSliceEmpty(brokers) {
		return nil, errors.New("brokers are empty or have an empty value")
	}

	if topic == "" {
		return nil, errors.New("topic is required")
	}

	return &Factory{name: name, topic: topic, brokers: brokers, saramaCfg: saramaCfg, oo: oo}, nil
}

func (c *consumer) OutOfOrder() bool {
	return false
}

// Create a new asynchronous consumer.
func (f *Factory) Create() (async.Consumer, error) {
	cc := kafka.ConsumerConfig{
		Brokers:      f.brokers,
		SaramaConfig: f.saramaCfg,
		Buffer:       f.saramaCfg.ChannelBufferSize,
	}

	c := &consumer{
		topic:  f.topic,
		config: cc,
	}
	c.partitions = c.partitionsFromOffset

	var err error
	for _, o := range f.oo {
		err = o(&c.config)
		if err != nil {
			return nil, err
		}
	}

	if c.config.DurationBasedConsumer {
		c.partitions = c.partitionsSinceDuration
	}

	if c.config.TimestampBasedConsumer {
		c.partitions = c.partitionsSinceTimestamp
	}

	if c.config.LatestOffsetReachedChan != nil {
		c.latestOffsetReachedChan = c.config.LatestOffsetReachedChan
	}

	return c, nil
}

// consumer members can be injected or overwritten with the usage of OptionFunc arguments.
type consumer struct {
	topic                   string
	cnl                     context.CancelFunc
	ms                      sarama.Consumer
	config                  kafka.ConsumerConfig
	partitions              func(context.Context) ([]sarama.PartitionConsumer, error)
	latestOffsetReachedChan chan<- struct{}
	latestOffsets           map[int32]int64
	// In the case the WithNotificationOnceReachingLatestOffset is used, we may end up in the case where the
	// starting offsets are equals to the latest offsets. In that situation, we don't want to wait for consuming
	// a first message before indicating we reached the end of this partition.
	startingOffsets map[int32]int64
	once            sync.Once
}

// Close handles closing consumer.
func (c *consumer) Close() error {
	if c.cnl != nil {
		c.cnl()
	}

	return nil
}

// Consume starts consuming messages from a Kafka topic.
func (c *consumer) Consume(ctx context.Context) (<-chan async.Message, <-chan error, error) {
	ctx, cnl := context.WithCancel(ctx)
	c.cnl = cnl

	chMsg := make(chan async.Message, c.config.Buffer)
	chErr := make(chan error, c.config.Buffer)

	log.Debugf("consuming messages from topic '%s' without using consumer group", c.topic)
	var pcs []sarama.PartitionConsumer

	pcs, err := c.partitions(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get partitions since duration: %w", err)
	}

	// When kafka cluster is not fully initialized, we may get 0 partitions.
	if len(pcs) == 0 {
		return nil, nil, errors.New("got 0 partitions")
	}

	var wg sync.WaitGroup
	if c.latestOffsetReachedChan != nil {
		wg.Add(len(pcs))
		go func() {
			// Wait for all the partition consumers to have reached the latest offset before closing the input channel.
			wg.Wait()
			// As the Consume function can be retried, we have to make sure the channel is closed only once.
			c.once.Do(func() {
				close(c.latestOffsetReachedChan)
			})
		}()
	}

	for i, pc := range pcs {
		var latestOffset int64
		if c.latestOffsetReachedChan != nil {
			latestOffset = c.latestOffsets[int32(i)]
		}
		go func(consumer sarama.PartitionConsumer, latestOffset int64, partition int) {
			latestOffsetReached := false
			if c.latestOffsetReachedChan != nil && c.ifStartingOffsetAfterLatestOffset(latestOffset, partition) {
				// In this case, we don't want to wait for consuming a message as we already know we're at the end of
				// the stream.
				latestOffsetReached = true
				wg.Done()
			}

			for {
				select {
				case <-ctx.Done():
					log.Info("canceling consuming messages requested")
					closePartitionConsumer(consumer)
					return
				case consumerError := <-consumer.Errors():
					closePartitionConsumer(consumer)
					chErr <- consumerError
					return
				case m := <-consumer.Messages():
					if c.latestOffsetReachedChan != nil && !latestOffsetReached && m.Offset >= latestOffset {
						latestOffsetReached = true
						wg.Done()
					}

					kafka.TopicPartitionOffsetDiffGaugeSet("", m.Topic, m.Partition, consumer.HighWaterMarkOffset(), m.Offset)
					kafka.MessageStatusCountInc(kafka.MessageReceived, "", m.Topic)

					msg, err := kafka.ClaimMessage(ctx, m, c.config.DecoderFunc, nil)
					if err != nil {
						kafka.MessageStatusCountInc(kafka.MessageClaimErrors, "", m.Topic)
						chErr <- err
						continue
					}
					kafka.MessageStatusCountInc(kafka.MessageDecoded, "", m.Topic)
					chMsg <- msg
				}
			}
		}(pc, latestOffset, i)
	}

	return chMsg, chErr, nil
}

func (c *consumer) ifStartingOffsetAfterLatestOffset(latestOffset int64, partition int) bool {
	return c.startingOffsets[int32(partition)] >= latestOffset
}

func (c *consumer) partitionsFromOffset(_ context.Context) ([]sarama.PartitionConsumer, error) {
	client, err := sarama.NewClient(c.config.Brokers, c.config.SaramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	ms, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("failed to create simple consumer: %w", err)
	}
	c.ms = ms

	partitions, err := c.ms.Partitions(c.topic)
	if err != nil {
		return nil, fmt.Errorf("failed to get partitions: %w", err)
	}

	pcs := make([]sarama.PartitionConsumer, len(partitions))

	for i, partition := range partitions {
		pc, err := c.ms.ConsumePartition(c.topic, partition, c.config.SaramaConfig.Consumer.Offsets.Initial)
		if nil != err {
			return nil, fmt.Errorf("failed to get partition consumer: %w", err)
		}
		pcs[i] = pc
	}

	if c.latestOffsetReachedChan != nil {
		err := c.setLatestOffsets(client, partitions)
		if err != nil {
			return nil, fmt.Errorf("failed to set latest offsets: %w", err)
		}

		err = c.setStartingOffsets(client, partitions)
		if err != nil {
			return nil, fmt.Errorf("failed to set starting offsets: %w", err)
		}
	}

	return pcs, nil
}

func (c *consumer) partitionsSinceDuration(ctx context.Context) ([]sarama.PartitionConsumer, error) {
	client, err := sarama.NewClient(c.config.Brokers, c.config.SaramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("failed to create simple consumer: %w", err)
	}
	c.ms = consumer

	durationKafkaClient, err := newDurationKafkaClient(client, consumer, c.config.SaramaConfig.Net.DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka duration client: %w", err)
	}

	durationClient, err := newDurationClient(durationKafkaClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create duration client: %w", err)
	}

	offsets, err := durationClient.getTimeBasedOffsetsPerPartition(ctx, c.topic, time.Now().Add(-c.config.DurationOffset), c.config.TimeExtractor)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve duration offsets per partition: %w", err)
	}
	c.startingOffsets = offsets

	partitions, err := c.ms.Partitions(c.topic)
	if err != nil {
		return nil, fmt.Errorf("failed to get partitions: %w", err)
	}

	pcs := make([]sarama.PartitionConsumer, len(partitions))

	for i, partition := range partitions {
		offset, exists := offsets[partition]
		if !exists {
			return nil, fmt.Errorf("partition %d unknown, this is most likely due to a repartitioning", partition)
		}

		pc, err := c.ms.ConsumePartition(c.topic, partition, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to get partition consumer: %w", err)
		}
		pcs[i] = pc
	}

	if c.latestOffsetReachedChan != nil {
		err := c.setLatestOffsets(client, partitions)
		if err != nil {
			return nil, fmt.Errorf("failed to set latest offsets: %w", err)
		}
	}

	return pcs, nil
}

func (c *consumer) partitionsSinceTimestamp(_ context.Context) ([]sarama.PartitionConsumer, error) {
	client, err := sarama.NewClient(c.config.Brokers, c.config.SaramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("failed to create simple consumer: %w", err)
	}
	c.ms = consumer

	partitions, err := c.ms.Partitions(c.topic)
	if err != nil {
		return nil, fmt.Errorf("failed to get partitions: %w", err)
	}

	pcs := make([]sarama.PartitionConsumer, len(partitions))

	ts := c.config.TimestampOffset
	c.startingOffsets = make(map[int32]int64, len(partitions))

	for i, partition := range partitions {
		offset, err := client.GetOffset(c.topic, partition, ts)
		if err != nil {
			return nil, fmt.Errorf("failed to get offset by timestamp %d for partition %d: %w", ts, partition, err)
		}
		c.startingOffsets[partition] = offset

		pc, err := c.ms.ConsumePartition(c.topic, partition, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to get partition consumer: %w", err)
		}
		pcs[i] = pc
	}

	if c.latestOffsetReachedChan != nil {
		err := c.setLatestOffsets(client, partitions)
		if err != nil {
			return nil, fmt.Errorf("failed to set latest offsets: %w", err)
		}
	}

	return pcs, nil
}

func (c *consumer) setLatestOffsets(client sarama.Client, partitions []int32) error {
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

func (c *consumer) setStartingOffsets(client sarama.Client, partitions []int32) error {
	offsets := make(map[int32]int64)
	for _, partitionID := range partitions {
		offset, err := client.GetOffset(c.topic, partitionID, c.config.SaramaConfig.Consumer.Offsets.Initial)
		if err != nil {
			return err
		}
		offsets[partitionID] = offset
	}
	c.startingOffsets = offsets
	return nil
}

func closePartitionConsumer(cns sarama.PartitionConsumer) {
	if cns == nil {
		return
	}
	err := cns.Close()
	if err != nil {
		log.Errorf("failed to close partition consumer: %v", err)
	}
}
