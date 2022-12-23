package main

import (
	"context"
	"os"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
)

const (
	name    = "example"
	version = "1.0.0"
)

func init() {
	err := os.Setenv("PATRON_LOG_LEVEL", "info")
	if err != nil {
		log.Fatalf("failed to set log level env var: %v", err)
	}
	err = os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "1.0")
	if err != nil {
		log.Fatalf("failed to set sampler env vars: %v", err)
	}
	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", examples.HTTPPort)
	if err != nil {
		log.Fatalf("failed to set default patron port env vars: %v", err)
	}
}

func main() {
	var options []patron.OptionFunc

	options = append(options, patron.WithTextLogger())

	// Setup HTTP
	router, err := createHttpRouter()
	if err != nil {
		log.Fatal(err)
	}

	options = append(options, patron.WithRouter(router))

	// Setup gRPC
	cmp, err := createGrpcServer()
	if err != nil {
		log.Fatal(err)
	}

	options = append(options, patron.WithComponents(cmp))

	// Setup Kafka
	cmp, err = createKafkaConsumer()
	if err != nil {
		log.Fatal(err)
	}

	options = append(options, patron.WithComponents(cmp))

	// Setup SQS
	cmp, err = createSQSConsumer()
	if err != nil {
		log.Fatal(err)
	}

	options = append(options, patron.WithComponents(cmp))

	// Setup AMQP
	cmp, err = createAMQPConsumer()
	if err != nil {
		log.Fatal(err)
	}

	options = append(options, patron.WithComponents(cmp))

	ctx := context.Background()

	service, err := patron.New(name, version, options...)
	if err != nil {
		log.Fatalf("failed to set up service: %v", err)
	}

	err = service.Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}
