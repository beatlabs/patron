package main

import (
	"context"
	"time"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/observability/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

func createKafkaConsumer() (patron.Component, error) {
	process := func(_ context.Context, records []*kgo.Record) error {
		for _, rec := range records {
			log.FromContext(rec.Context).Info("kafka message received", "msg", string(rec.Value))
		}
		return nil
	}

	return kafka.New(name, examples.KafkaGroup, []string{examples.KafkaBroker}, []string{examples.KafkaTopic}, process, nil,
		kafka.WithFailureStrategy(kafka.SkipStrategy), kafka.WithBatchSize(1), kafka.WithBatchTimeout(1*time.Second),
		kafka.WithRetries(10), kafka.WithRetryWait(3*time.Second), kafka.WithManualCommit())
}
