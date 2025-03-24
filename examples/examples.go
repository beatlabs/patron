package examples

import (
	"context"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
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

type SQSCustomResolver struct{}

func (cr *SQSCustomResolver) ResolveEndpoint(_ context.Context, _ sqs.EndpointParameters) (smithyendpoints.Endpoint, error) {
	uri, err := url.Parse(AWSSQSEndpoint)
	if err != nil {
		return smithyendpoints.Endpoint{}, err
	}
	return smithyendpoints.Endpoint{
		URI: *uri,
	}, nil
}

func CreateSQSConfig(ctx context.Context) (aws.Config, error) {
	return config.LoadDefaultConfig(ctx,
		config.WithRegion(AWSRegion),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(AWSID, AWSSecret, AWSToken))),
	)
}
