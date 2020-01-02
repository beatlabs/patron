package simple

import (
	"context"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/async/kafka"
	"github.com/beatlabs/patron/errors"
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

	if len(brokers) == 0 {
		return nil, errors.New("provide at least one broker")
	}

	if topic == "" {
		return nil, errors.New("topic is required")
	}

	return &Factory{name: name, topic: topic, brokers: brokers, oo: oo}, nil
}

// Create a new consumer.
func (f *Factory) Create() (async.Consumer, error) {

	config, err := kafka.SaramaConfig(f.name)

	if err != nil {
		return nil, err
	}

	cc := kafka.ConsumerConfig{
		Brokers:      f.brokers,
		Buffer:       1000,
		SaramaConfig: config,
	}

	c := &consumer{
		topic:       f.topic,
		consumerCnf: cc,
	}

	for _, o := range f.oo {
		err = o(c)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

// consumer members can be injected or overwritten with the usage of OptionFunc arguments.
type consumer struct {
	kafka.Consumer
	topic       string
	cnl         context.CancelFunc
	ms          sarama.Consumer
	consumerCnf kafka.ConsumerConfig
}

func (c *consumer) ConsumerConfig() *kafka.ConsumerConfig { return &c.consumerCnf }

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

	return consume(ctx, c)
}

func consume(ctx context.Context, c *consumer) (<-chan async.Message, <-chan error, error) {

	chMsg := make(chan async.Message, c.ConsumerConfig().Buffer)
	chErr := make(chan error, c.ConsumerConfig().Buffer)

	log.Infof("consuming messages from topic '%s' without using consumer group", c.topic)
	pcs, err := c.partitions()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get partitions")
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

					go func() {
						msg, err := kafka.ClaimMessage(ctx, m, c.ConsumerConfig().DecoderFunc, nil)
						if err != nil {
							chErr <- err
							return
						}
						chMsg <- msg
					}()
				}
			}
		}(pc)
	}

	return chMsg, chErr, nil
}

func (c *consumer) partitions() ([]sarama.PartitionConsumer, error) {

	ms, err := sarama.NewConsumer(c.ConsumerConfig().Brokers, c.ConsumerConfig().SaramaConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create consumer")
	}
	c.ms = ms

	partitions, err := c.ms.Partitions(c.topic)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get partitions")
	}

	pcs := make([]sarama.PartitionConsumer, len(partitions))

	for i, partition := range partitions {

		pc, err := c.ms.ConsumePartition(c.topic, partition, c.ConsumerConfig().SaramaConfig.Consumer.Offsets.Initial)
		if nil != err {
			return nil, errors.Wrap(err, "failed to get partition consumer")
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
