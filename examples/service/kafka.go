package main

import (
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/component/kafka/group"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
)

func createKafkaConsumer() (patron.Component, error) {
	cfg, err := kafka.DefaultConsumerSaramaConfig("kafka-consumer", true)
	if err != nil {
		return nil, err
	}

	cfg.Version = sarama.V2_6_0_0

	process := func(batch kafka.Batch) error {
		for _, msg := range batch.Messages() {
			log.FromContext(msg.Context()).Info("kafka message received: %s", string(msg.Message().Value))
		}
		return nil
	}

	return group.New(name, examples.KafkaGroup, []string{examples.KafkaBroker}, []string{examples.KafkaTopic}, process, cfg,
		group.WithFailureStrategy(kafka.SkipStrategy), group.WithBatchSize(1), group.WithBatchTimeout(1*time.Second),
		group.WithRetries(10), group.WithRetryWait(3*time.Second), group.WithCommitSync())
}
