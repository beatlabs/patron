package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/beatlabs/patron"
	patronamqp "github.com/beatlabs/patron/client/amqp/v2"
	"github.com/beatlabs/patron/component/async"
	"github.com/beatlabs/patron/component/async/kafka"
	"github.com/beatlabs/patron/component/async/kafka/group"
	kafkacmp "github.com/beatlabs/patron/component/kafka"
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
		log.Fatalf("failed to set log level env var: %v", err)
	}
	err = os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "1.0")
	if err != nil {
		log.Fatalf("failed to set sampler env vars: %v", err)
	}
	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", "50002")
	if err != nil {
		log.Fatalf("failed to set default patron port env vars: %v", err)
	}
}

func main() {
	name := "kafka"
	version := "1.0.0"

	service, err := patron.New(name, version, patron.TextLogger())
	if err != nil {
		log.Fatalf("failed to set up service: %v", err)
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

	saramaCfg, err := kafkacmp.DefaultConsumerSaramaConfig("kafka-legacy", false)
	if err != nil {
		return nil, err
	}

	cf, err := group.New(name, groupID, []string{topic}, []string{broker}, saramaCfg, kafka.Decoder(json.DecodeRaw))
	if err != nil {
		return nil, err
	}

	cmp, err := async.New("kafka-cmp", cf, kafkaCmp.Process).WithRetries(10).
		WithRetryWait(5 * time.Second).Create()
	if err != nil {
		return nil, err
	}
	kafkaCmp.cmp = cmp

	return &kafkaCmp, nil
}

func (kc *kafkaComponent) Process(msg async.Message) error {
	var u examples.User

	err := msg.Decode(&u)
	if err != nil {
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
	return nil
}
