// Package examples provides example code for the patron framework.
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
	// HTTPPort is the port of the HTTP service.
	HTTPPort = "50001"
	// HTTPURL is the URL of the HTTP service.
	HTTPURL  = "http://localhost:50001"

	// GRPCPort is the port of the gRPC service.
	GRPCPort   = "50002"
	// GRPCTarget is the target of the gRPC service.
	GRPCTarget = "localhost:50002"

	// AMQPURL is the URL of the AMQP broker.
	AMQPURL          = "amqp://bitnami:bitnami@localhost:5672/" //nolint:gosec
	// AMQPQueue is the name of the AMQP queue.
	AMQPQueue        = "patron"
	// AMQPExchangeName is the name of the AMQP exchange.
	AMQPExchangeName = "patron"
	// AMQPExchangeType is the type of the AMQP exchange.
	AMQPExchangeType = amqp.ExchangeFanout

	// AWSRegion is the AWS region.
	AWSRegion      = "eu-west-1"
	// AWSID is the AWS ID.
	AWSID          = "test"
	// AWSSecret is the AWS secret.
	AWSSecret      = "test"
	// AWSToken is the AWS token.
	AWSToken       = "token"
	// AWSSQSEndpoint is the SQS endpoint.
	AWSSQSEndpoint = "http://localhost:4566"
	// AWSSQSQueue is the SQS queue name.
	AWSSQSQueue    = "patron"

	// KafkaTopic is the Kafka topic.
	KafkaTopic  = "patron-topic"
	// KafkaGroup is the Kafka consumer group.
	KafkaGroup  = "patron-group"
	// KafkaBroker is the Kafka broker address.
	KafkaBroker = "localhost:9092"
)

// SQSCustomResolver is a custom SQS endpoint resolver.
type SQSCustomResolver struct{}

// ResolveEndpoint resolves the endpoint for SQS.
func (cr *SQSCustomResolver) ResolveEndpoint(_ context.Context, _ sqs.EndpointParameters) (smithyendpoints.Endpoint, error) {
	uri, err := url.Parse(AWSSQSEndpoint)
	if err != nil {
		return smithyendpoints.Endpoint{}, err
	}
	return smithyendpoints.Endpoint{
		URI: *uri,
	}, nil
}

// CreateSQSConfig creates an AWS config for SQS.
func CreateSQSConfig(ctx context.Context) (aws.Config, error) {
	return config.LoadDefaultConfig(ctx,
		config.WithRegion(AWSRegion),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(AWSID, AWSSecret, AWSToken))),
	)
}
