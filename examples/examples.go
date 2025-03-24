package examples

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	HTTPPort = "50001"
	HTTPURL  = "http://localhost:50001"

	GRPCPort   = "50002"
	GRPCTarget = "localhost:50002"

	AMQPURL          = "amqp://bitnami:bitnami@localhost:5672/" //nolint:gosec
	AMQPQueue        = "patron"
	AMQPExchangeName = "patron"
	AMQPExchangeType = amqp.ExchangeFanout

	AWSRegion      = "eu-west-1"
	AWSID          = "test"
	AWSSecret      = "test"
	AWSToken       = "token"
	AWSSQSEndpoint = "http://localhost:4566"
	AWSSQSQueue    = "patron"

	KafkaTopic  = "patron-topic"
	KafkaGroup  = "patron-group"
	KafkaBroker = "localhost:9092"
)

func CreateSQSConfig(ctx context.Context) (aws.Config, error) {
	return config.LoadDefaultConfig(ctx,
		config.WithRegion(AWSRegion),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(AWSID, AWSSecret, AWSToken))),
	)
}
