package group

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/async/kafka"
	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/opentracing/opentracing-go"
)

// Factory definition of a consumer factory.
type Factory struct {
	name    string
	group   string
	topic   string
	brokers []string
	oo      []kafka.OptionFunc
}

// New constructor.
func New(name, group, topic string, brokers []string, oo ...kafka.OptionFunc) (*Factory, error) {

	if name == "" {
		return nil, errors.New("name is required")
	}

	if group == "" {
		return nil, errors.New("group is required")
	}

	if len(brokers) == 0 {
		return nil, errors.New("provide at least one broker")
	}

	if topic == "" {
		return nil, errors.New("topic is required")
	}

	return &Factory{name: name, group: group, topic: topic, brokers: brokers, oo: oo}, nil
}

// Create a new consumer.
func (f *Factory) Create() (async.Consumer, error) {

	config, err := kafka.SaramaConfig(f.name)

	if err != nil {
		return nil, err
	}

	cc := kafka.ConsumerConfig{
		Brokers:      f.brokers,
		Buffer:       0,
		SaramaConfig: config,
	}

	c := &consumer{
		topic:       f.topic,
		group:       f.group,
		traceTag:    opentracing.Tag{Key: "group", Value: f.group},
		consumerCnf: cc,
	}

	for _, o := range f.oo {
		err = o(c)
		if err != nil {
			return nil, fmt.Errorf("Could not apply OptionFunc to consumer : %v", err)
		}
	}

	return c, nil
}

// consumer members can be injected or overwritten with the usage of OptionFunc arguments.
type consumer struct {
	kafka.Consumer
	topic       string
	group       string
	traceTag    opentracing.Tag
	cnl         context.CancelFunc
	cg          sarama.ConsumerGroup
	consumerCnf kafka.ConsumerConfig
}

func (c *consumer) ConsumerConfig() *kafka.ConsumerConfig { return &c.consumerCnf }

// Close handles closing consumer.
func (c *consumer) Close() error {
	if c.cnl != nil {
		c.cnl()
	}

	return errors.Wrap(c.cg.Close(), "failed to close consumer")
}

// Consume starts consuming messages from a Kafka topic.
func (c *consumer) Consume(ctx context.Context) (<-chan async.Message, <-chan error, error) {
	ctx, cnl := context.WithCancel(ctx)
	c.cnl = cnl

	cg, err := sarama.NewConsumerGroup(c.ConsumerConfig().Brokers, c.group, c.ConsumerConfig().SaramaConfig)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create consumer")
	}
	c.cg = cg
	log.Infof("consuming messages from topic '%s' using group '%s'", c.topic, c.group)

	chMsg := make(chan async.Message, c.ConsumerConfig().Buffer)
	chErr := make(chan error, c.ConsumerConfig().Buffer)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info("canceling consuming messages requested")
				closeConsumer(c.cg)
				return
			case consumerError := <-c.cg.Errors():
				closeConsumer(c.cg)
				chErr <- consumerError
				return
			}
		}
	}()

	// Iterate over consumer sessions.
	go func() {
		hnd := handler{consumer: c, messages: chMsg}
		for {
			err := c.cg.Consume(ctx, []string{c.topic}, hnd)
			if err != nil {
				chErr <- err
			}
		}
	}()

	return chMsg, chErr, nil
}

func closeConsumer(cns sarama.ConsumerGroup) {
	if cns == nil {
		return
	}
	err := cns.Close()
	if err != nil {
		log.Errorf("failed to close partition consumer: %v", err)
	}
}

type handler struct {
	consumer *consumer
	messages chan async.Message
}

func (h handler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h handler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (h handler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	ctx := sess.Context()
	for msg := range claim.Messages() {
		kafka.TopicPartitionOffsetDiffGaugeSet(h.consumer.group, msg.Topic, msg.Partition, claim.HighWaterMarkOffset(), msg.Offset)
		m, err := kafka.ClaimMessage(ctx, msg, h.consumer.ConsumerConfig().DecoderFunc, sess)
		if err != nil {
			return err
		}
		h.messages <- m
	}
	return nil
}
