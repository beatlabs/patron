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
	err := os.Setenv("PATRON_LOG_LEVEL", "debug")
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
	ctx := context.Background()

	service, err := patron.New(name, version, patron.WithTextLogger())
	if err != nil {
		log.Fatalf("failed to set up service: %v", err)
	}

	var components []patron.Component

	// Setup HTTP
	cmp, err := createHttpRouter()
	if err != nil {
		log.Fatal(err)
	}

	components = append(components, cmp)

	// Setup gRPC
	cmp, err = createGrpcServer()
	if err != nil {
		log.Fatal(err)
	}

	components = append(components, cmp)

	// Setup Kafka
	cmp, err = createKafkaConsumer()
	if err != nil {
		log.Fatal(err)
	}

	components = append(components, cmp)

	// Setup SQS
	cmp, err = createSQSConsumer()
	if err != nil {
		log.Fatal(err)
	}

	components = append(components, cmp)

	// Setup AMQP
	cmp, err = createAMQPConsumer()
	if err != nil {
		log.Fatal(err)
	}

	components = append(components, cmp)

	err = service.Run(ctx, components...)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}
