package main

import (
	"fmt"
	"os"
	"time"

	"github.com/thebeatapp/patron"
	"github.com/thebeatapp/patron/async"
	"github.com/thebeatapp/patron/async/kafka"
	"github.com/thebeatapp/patron/encoding/json"
	"github.com/thebeatapp/patron/examples"
	"github.com/thebeatapp/patron/log"
	"github.com/thebeatapp/patron/trace/amqp"
)

const (
	amqpURL      = "amqp://guest:guest@localhost:5672/"
	amqpExchange = "patron"
	amqpQueue    = "patron"
	kafkaTopic   = "patron-topic"
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

	kafkaCmp, err := newKafkaComponent(name, kafkaBroker, kafkaTopic, amqpURL, amqpExchange)
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

	err = srv.Run()
	if err != nil {
		log.Fatalf("failed to run service %v", err)
	}
}

type kafkaComponent struct {
	cmp patron.Component
	pub amqp.Publisher
}

func newKafkaComponent(name, broker, topic, amqpURL, amqpExc string) (*kafkaComponent, error) {

	kafkaCmp := kafkaComponent{}

	cf, err := kafka.New(name, json.Type, topic, []string{broker})
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
