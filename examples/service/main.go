package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/observability/log"
)

const (
	name    = "example"
	version = "1.0.0"
)

func init() {
	err := os.Setenv("PATRON_LOG_LEVEL", "debug")
	if err != nil {
		slog.Error("failed to set log level env var", log.ErrorAttr(err))
		os.Exit(1)
	}
	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", examples.HTTPPort)
	if err != nil {
		slog.Error("failed to set default patron port env vars", log.ErrorAttr(err))
		os.Exit(1)
	}
}

func main() {
	ctx := context.Background()

	service, err := patron.New(name, version)
	if err != nil {
		slog.Error("failed to set up service", log.ErrorAttr(err))
		os.Exit(1)
	}

	var components []patron.Component

	// Setup HTTP
	cmp, err := createHTTPRouter()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	components = append(components, cmp)

	// Setup gRPC
	cmp, err = createGrpcServer()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	components = append(components, cmp)

	// Setup Kafka
	cmp, err = createKafkaConsumer()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	components = append(components, cmp)

	// Setup SQS
	cmp, err = createSQSConsumer(ctx)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	components = append(components, cmp)

	// Setup AMQP
	cmp, err = createAMQPConsumer()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	components = append(components, cmp)

	err = service.Run(ctx, components...)
	if err != nil {
		slog.Error("failed to create and run service", log.ErrorAttr(err))
		os.Exit(1)
	}
}
