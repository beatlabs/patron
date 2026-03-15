package main

import (
	"time"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/observability/log"
)

func createKafkaConsumer() (patron.Component, error) {
	opts, err := kafka.DefaultConsumerConfig("kafka-consumer", true)
	if err != nil {
		return nil, err
	}

	process := func(batch kafka.Batch) error {
		for _, msg := range batch.Messages() {
			log.FromContext(msg.Context()).Info("kafka message received", "msg", string(msg.Record().Value))
			continue
		}
		return nil
	}

	return kafka.New(name, examples.KafkaGroup, []string{examples.KafkaBroker}, []string{examples.KafkaTopic}, process, opts,
		kafka.WithFailureStrategy(kafka.SkipStrategy), kafka.WithBatchSize(1), kafka.WithBatchTimeout(1*time.Second),
		kafka.WithRetries(10), kafka.WithRetryWait(3*time.Second), kafka.WithCommitSync())
}
