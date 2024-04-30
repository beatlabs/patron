package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/examples"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	name    = "example"
	version = "1.0.0"
)

func init() {
	err := os.Setenv("PATRON_LOG_LEVEL", "debug")
	if err != nil {
		slog.Error("failed to set log level env var", slog.Any("error", err))
		os.Exit(1)
	}
	err = os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "1.0")
	if err != nil {
		slog.Error("failed to set sampler env vars", slog.Any("error", err))
		os.Exit(1)
	}
	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", examples.HTTPPort)
	if err != nil {
		slog.Error("failed to set default patron port env vars", slog.Any("error", err))
		os.Exit(1)
	}
}

func main() {
	ctx := context.Background()

	conn, err := grpc.NewClient("localhost:4317", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("failed to create grpc observability client", slog.Any("error", err))
		os.Exit(1)
	}

	service, err := patron.New(name, version, conn)
	if err != nil {
		slog.Error("failed to set up service", slog.Any("error", err))
		os.Exit(1)
	}

	var components []patron.Component

	// Setup HTTP
	cmp, err := createHttpRouter()
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
	cmp, err = createSQSConsumer()
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
		slog.Error("failed to create and run service", slog.Any("error", err))
		os.Exit(1)
	}
}
