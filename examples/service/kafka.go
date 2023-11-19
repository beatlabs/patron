package main

import (
	"time"

	"github.com/IBM/sarama"
	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/kafka"
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

	return kafka.New(name, examples.KafkaGroup, []string{examples.KafkaBroker}, []string{examples.KafkaTopic}, process, cfg,
		kafka.WithFailureStrategy(kafka.SkipStrategy), kafka.WithBatchSize(1), kafka.WithBatchTimeout(1*time.Second),
		kafka.WithRetries(10), kafka.WithRetryWait(3*time.Second), kafka.WithCommitSync())
}
