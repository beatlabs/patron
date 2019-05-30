package main

import (
	"fmt"
	"os"
	"time"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/async"
	"github.com/beatlabs/patron/async/amqp"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	oamqp "github.com/streadway/amqp"
)

const (
	amqpURL          = "amqp://guest:guest@localhost:5672/"
	amqpQueue        = "patron"
	amqpExchangeName = "patron"
	amqpExchangeType = oamqp.ExchangeDirect
)

var (
	amqpBindings = []string{"bind.one.*", "bind.two.*"}
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
	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", "50003")
	if err != nil {
		fmt.Printf("failed to set default patron port env vars: %v", err)
		os.Exit(1)
	}
}

func main() {
	name := "fourth"
	version := "1.0.0"

	err := patron.Setup(name, version)
	if err != nil {
		fmt.Printf("failed to set up logging: %v", err)
		os.Exit(1)
	}

	amqpCmp, err := newAmqpComponent(amqpURL, amqpQueue, amqpExchangeName, amqpExchangeType, amqpBindings)
	if err != nil {
		log.Fatalf("failed to create processor %v", err)
	}

	srv, err := patron.New(
		name,
		version,
		patron.Components(amqpCmp.cmp),
	)
	if err != nil {
		log.Fatalf("failed to create service %v", err)
	}

	err = srv.Run()
	if err != nil {
		log.Fatalf("failed to run service %v", err)
	}
}

type amqpComponent struct {
	cmp patron.Component
}

func newAmqpComponent(url, queue, exchangeName, exchangeType string, bindings []string) (*amqpComponent, error) {

	amqpCmp := amqpComponent{}

	exchange, err := amqp.NewExchange(exchangeName, exchangeType)

	if err != nil {
		return nil, err
	}

	cf, err := amqp.New(url, queue, *exchange, amqp.Bindings(bindings...))
	if err != nil {
		return nil, err
	}

	cmp, err := async.New("amqp-cmp", amqpCmp.Process, cf, async.ConsumerRetry(10, 10*time.Second))
	if err != nil {
		return nil, err
	}
	amqpCmp.cmp = cmp

	return &amqpCmp, nil
}

func (ac *amqpComponent) Process(msg async.Message) error {
	var u examples.User

	err := msg.Decode(&u)
	if err != nil {
		return err
	}

	log.FromContext(msg.Context()).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return nil
}
