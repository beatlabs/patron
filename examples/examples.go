package examples

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	HTTPPort = "50001"
	HTTPURL  = "http://localhost:50001"

	GRPCPort   = "50002"
	GRPCTarget = "localhost:50002"

	AMQPURL          = "amqp://bitnami:bitnami@localhost:5672/"
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

func CreateSQSConfig() (aws.Config, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == sqs.ServiceID && region == AWSRegion {
			return aws.Endpoint{
				URL:           AWSSQSEndpoint,
				SigningRegion: AWSRegion,
			}, nil
		}
		// returning EndpointNotFoundError will allow the service to fallback to it's default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	return config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(AWSRegion),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(AWSID, AWSSecret, AWSToken))),
	)
}
