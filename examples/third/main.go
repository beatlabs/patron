package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/async/kafka"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace/amqp"
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
	name := "third"
	version := "1.0.0"

	err := patron.Setup(name, version)
	if err != nil {
		fmt.Printf("failed to set up logging: %v", err)
		os.Exit(1)
	}

	kafkaCmp, err := newKafkaComponent(name, kafkaBroker, kafkaTopic, kafkaGroup, amqpURL, amqpExchange)
	if err != nil {
		log.Fatalf("failed to create processor %v", err)
	}

	srv, err := patron.New(
		name,
		version,
		patron.Components(kafkaCmp.cmp),
	)
	if err != nil {
		log.Fatalf("failed to create service %v", err)
	}

	ctx := context.Background()
	err = srv.Run(ctx)
	if err != nil {
		log.Fatalf("failed to run service %v", err)
	}
}

type kafkaComponent struct {
	cmp patron.Component
	pub amqp.Publisher
}

func newKafkaComponent(name, broker, topic, group, amqpURL, amqpExc string) (*kafkaComponent, error) {

	kafkaCmp := kafkaComponent{}

	cf, err := kafka.New(name, topic, group, []string{broker}, kafka.Decoder(json.DecodeRaw))
	if err != nil {
		return nil, err
	}

	cmp, err := async.New("kafka-cmp", kafkaCmp.Process, cf, async.ConsumerRetry(10, 5*time.Second))
	if err != nil {
		return nil, err
	}
	kafkaCmp.cmp = cmp

	pub, err := amqp.NewPublisher(amqpURL, amqpExc)
	if err != nil {
		return nil, err
	}
	kafkaCmp.pub = pub

	return &kafkaCmp, nil
}

func (kc *kafkaComponent) Process(msg async.Message) error {
	var u examples.User

	err := msg.Decode(&u)
	if err != nil {
		return err
	}

	amqpMsg, err := amqp.NewProtobufMessage(&u)
	if err != nil {
		return err
	}

	err = kc.pub.Publish(msg.Context(), amqpMsg)
	if err != nil {
		return err
	}

	log.FromContext(msg.Context()).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return nil
}
