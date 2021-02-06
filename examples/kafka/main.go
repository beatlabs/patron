package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron/component/kafka"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/log"
)

const (
	kafkaTopic  = "patron-topic"
	kafkaGroup  = "patron-group"
	kafkaBroker = "localhost:9092"
)

func init() {
	err := os.Setenv("PATRON_LOG_LEVEL", "debug")
	if err != nil {
		fmt.Printf("failed to set log level env var: %v", err)
		os.Exit(1)
	}
	err = os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "1.0")
	if err != nil {
		fmt.Printf("failed to set sampler env vars: %v", err)
		os.Exit(1)
	}
	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", "50002")
	if err != nil {
		fmt.Printf("failed to set default patron port env vars: %v", err)
		os.Exit(1)
	}
}

func main() {
	name := "kafka"
	version := "1.0.0"

	service, err := patron.New(name, version, patron.TextLogger())
	if err != nil {
		fmt.Printf("failed to set up service: %v", err)
		os.Exit(1)
	}

	kafkaCmp, err := newKafkaComponent(name, kafkaBroker, kafkaTopic, kafkaGroup)
	if err != nil {
		log.Fatalf("failed to create processor %v", err)
	}

	ctx := context.Background()
	err = service.WithComponents(kafkaCmp.cmp).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}

type kafkaComponent struct {
	cmp patron.Component
}

func newKafkaComponent(name, broker, topic, groupID string) (*kafkaComponent, error) {
	kafkaCmp := kafkaComponent{}

	saramaCfg := sarama.NewConfig()
	// consumer will use WithSyncCommit so will manually commit offsets
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = false
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaCfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky
	saramaCfg.Net.DialTimeout = 15 * time.Second

	cmp, err := kafka.New(name, groupID, []string{broker}, []string{topic}, kafkaCmp.Process).
		WithBatching(1, 1*time.Second).
		WithFailureStrategy(kafka.SkipStrategy).
		WithRetries(3).
		WithRetryWait(1 * time.Second).
		WithSyncCommit().
		WithSaramaConfig(saramaCfg).
		Create()

	if err != nil {
		return nil, err
	}
	kafkaCmp.cmp = cmp

	return &kafkaCmp, nil
}

func (kc *kafkaComponent) Process(ctx context.Context, msgs []kafka.MessageWrapper) error {
	log.FromContext(ctx).Infof("got batch: %+v", msgs)
	return nil
}
