// Package simple provides a simple consumer implementation without consumer groups.
package simple

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/component/async/kafka"
	"github.com/beatlabs/patron/internal/validation"
	"github.com/beatlabs/patron/log"
)

// Factory definition of a consumer factory.
type Factory struct {
	name    string
	topic   string
	brokers []string
	oo      []kafka.OptionFunc
}

// New constructor.
func New(name, topic string, brokers []string, oo ...kafka.OptionFunc) (*Factory, error) {

	if name == "" {
		return nil, errors.New("name is required")
	}

	if validation.IsStringSliceEmpty(brokers) {
		return nil, errors.New("brokers are empty or have an empty value")
	}

	if topic == "" {
		return nil, errors.New("topic is required")
	}

	return &Factory{name: name, topic: topic, brokers: brokers, oo: oo}, nil
}

func (c *consumer) OutOfOrder() bool {
	return false
}

// Create a new consumer.
func (f *Factory) Create() (async.Consumer, error) {

	config, err := kafka.DefaultSaramaConfig(f.name)
	if err != nil {
		return nil, err
	}

	cc := kafka.ConsumerConfig{
		Brokers:      f.brokers,
		Buffer:       1000,
		SaramaConfig: config,
	}

	c := &consumer{
		topic:  f.topic,
		config: cc,
	}
	c.partitions = c.partitionsFromOffset

	for _, o := range f.oo {
		err = o(&c.config)
		if err != nil {
			return nil, err
		}
	}

	if c.config.DurationBasedConsumer {
		c.partitions = c.partitionsSinceDuration
	}

	return c, nil
}

// consumer members can be injected or overwritten with the usage of OptionFunc arguments.
type consumer struct {
	topic      string
	cnl        context.CancelFunc
	ms         sarama.Consumer
	config     kafka.ConsumerConfig
	partitions func(context.Context) ([]sarama.PartitionConsumer, error)
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

	log.Infof("consuming messages from topic '%s' without using consumer group", c.topic)
	var pcs []sarama.PartitionConsumer

	pcs, err := c.partitions(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get partitions since duration: %w", err)
	}

	// When kafka cluster is not fully initialized, we may get 0 partitions.
	if len(pcs) == 0 {
		return nil, nil, errors.New("got 0 partitions")
	}

	for _, pc := range pcs {
		go func(consumer sarama.PartitionConsumer) {
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
		}(pc)
	}

	return chMsg, chErr, nil
}

func (c *consumer) partitionsFromOffset(_ context.Context) ([]sarama.PartitionConsumer, error) {

	ms, err := sarama.NewConsumer(c.config.Brokers, c.config.SaramaConfig)
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

	return pcs, nil
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
