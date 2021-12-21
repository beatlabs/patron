// Package group provides a consumer group implementation.
package group

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/component/async/kafka"
	"github.com/beatlabs/patron/internal/validation"
	"github.com/beatlabs/patron/log"
	opentracing "github.com/opentracing/opentracing-go"
)

// Factory definition of a consumer factory.
type Factory struct {
	name    string
	group   string
	topics  []string
	brokers []string
	cfg     *sarama.Config
	oo      []kafka.OptionFunc
}

// New constructor.
func New(name, group string, topics, brokers []string, saramaCfg *sarama.Config, oo ...kafka.OptionFunc) (*Factory, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}

	if group == "" {
		return nil, errors.New("group is required")
	}

	if saramaCfg == nil {
		return nil, errors.New("no Sarama configuration specified")
	}

	if validation.IsStringSliceEmpty(brokers) {
		return nil, errors.New("brokers are empty or have an empty value")
	}

	if validation.IsStringSliceEmpty(topics) {
		return nil, errors.New("topics are empty or have an empty value")
	}

	return &Factory{name: name, group: group, topics: topics, brokers: brokers, cfg: saramaCfg, oo: oo}, nil
}

func (c *consumer) OutOfOrder() bool {
	return false
}

// Create a new asynchronous group consumer.
func (f *Factory) Create() (async.Consumer, error) {
	cc := kafka.ConsumerConfig{
		Brokers:      f.brokers,
		SaramaConfig: f.cfg,
		Buffer:       f.cfg.ChannelBufferSize,
	}

	c := &consumer{
		topics:   f.topics,
		group:    f.group,
		traceTag: opentracing.Tag{Key: "group", Value: f.group},
		config:   cc,
	}

	var err error
	for _, o := range f.oo {
		err = o(&c.config)
		if err != nil {
			return nil, fmt.Errorf("could not apply OptionFunc to consumer : %w", err)
		}
	}

	return c, nil
}

// consumer members can be injected or overwritten with the usage of OptionFunc arguments.
type consumer struct {
	topics   []string
	group    string
	traceTag opentracing.Tag
	cnl      context.CancelFunc
	cg       sarama.ConsumerGroup
	config   kafka.ConsumerConfig
}

// Close handles closing consumer.
func (c *consumer) Close() error {
	if c.cnl != nil {
		c.cnl()
	}

	if c.cg != nil {
		err := c.cg.Close()
		if err != nil {
			return fmt.Errorf("failed to close consumer: %w", err)
		}
	}
	return nil
}

// Consume starts consuming messages from a Kafka topic.
func (c *consumer) Consume(ctx context.Context) (<-chan async.Message, <-chan error, error) {
	ctx, cnl := context.WithCancel(ctx)
	c.cnl = cnl

	cg, err := sarama.NewConsumerGroup(c.config.Brokers, c.group, c.config.SaramaConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create consumer: %w", err)
	}
	c.cg = cg
	log.Debugf("consuming messages from topics '%s' using group '%s'", strings.Join(c.topics, ","), c.group)

	chMsg := make(chan async.Message, c.config.Buffer)
	chErr := make(chan error, c.config.Buffer)

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info("canceling consuming messages requested")
				closeConsumer(c.cg)
				return
			case consumerError := <-c.cg.Errors():
				chErr <- consumerError
				closeConsumer(c.cg)
				return
			}
		}
	}()

	// Iterate over consumer sessions.
	go func() {
		hnd := handler{consumer: c, messages: chMsg}
		for {
			err := c.cg.Consume(ctx, c.topics, hnd)
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
		log.Errorf("failed to close consumer group: %v", err)
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
		kafka.MessageStatusCountInc(kafka.MessageReceived, h.consumer.group, msg.Topic)

		m, err := kafka.ClaimMessage(ctx, msg, h.consumer.config.DecoderFunc, sess)
		if err != nil {
			kafka.MessageStatusCountInc(kafka.MessageClaimErrors, h.consumer.group, msg.Topic)
			return err
		}
		kafka.MessageStatusCountInc(kafka.MessageDecoded, h.consumer.group, msg.Topic)
		h.messages <- m
	}
	return nil
}
