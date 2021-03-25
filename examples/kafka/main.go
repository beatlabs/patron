package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Shopify/sarama"
	"github.com/beatlabs/patron"
	patronamqp "github.com/beatlabs/patron/client/amqp/v2"
	"github.com/beatlabs/patron/component/kafka"
	"github.com/beatlabs/patron/component/kafka/group"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	"github.com/streadway/amqp"
)

const (
	amqpURL      = "amqp://guest:guest@localhost:5672/"
	amqpExchange = "patron"
	kafkaTopic   = "patron-topic"
	kafkaGroup   = "patron-group"
	kafkaBroker  = "localhost:9092"
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

	pub, err := patronamqp.New(amqpURL)
	if err != nil {
		log.Fatalf("failed to create AMQP publisher processor %v", err)
	}
	defer func() {
		if err := pub.Close(); err != nil {
			log.Errorf("failed to close AMQP publisher: %v", err)
		}
	}()

	kafkaCmp, err := newKafkaComponent(name, kafkaBroker, kafkaTopic, kafkaGroup, pub)
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
	pub *patronamqp.Publisher
}

func newKafkaComponent(name, broker, topic, groupID string, publisher *patronamqp.Publisher) (*kafkaComponent, error) {
	kafkaCmp := kafkaComponent{
		pub: publisher,
	}

	saramaCfg := sarama.NewConfig()
	// batches will be responsible for committing
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = false
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaCfg.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky
	saramaCfg.Net.DialTimeout = 15 * time.Second
	saramaCfg.Version = sarama.V2_6_0_0

	cmp, err := group.New(
		name,
		groupID,
		[]string{broker},
		[]string{topic},
		kafkaCmp.Process,
		group.FailureStrategy(kafka.SkipStrategy),
		group.BatchSize(1),
		group.BatchTimeout(1*time.Second),
		group.Retries(10),
		group.RetryWait(3*time.Second),
		group.SaramaConfig(saramaCfg),
		group.CommitSync())

	if err != nil {
		return nil, err
	}
	kafkaCmp.cmp = cmp

	return &kafkaCmp, nil
}

func (kc *kafkaComponent) Process(batch kafka.Batch) error {
	for _, msg := range batch.Messages() {
		var u examples.User
		err := json.DecodeRaw(msg.Message().Value, &u)
		if err != nil {
			log.FromContext(msg.Context()).Errorf("error decoding kafka message: %w", err)
			return err
		}

		body, err := protobuf.Encode(&u)
		if err != nil {
			return fmt.Errorf("failed to encode to protobuf: %w", err)
		}

		amqpMsg := amqp.Publishing{
			ContentType: protobuf.Type,
			Body:        body,
		}

		err = kc.pub.Publish(msg.Context(), amqpExchange, "", false, false, amqpMsg)
		if err != nil {
			return err
		}

		log.FromContext(msg.Context()).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	}
	return nil
}
