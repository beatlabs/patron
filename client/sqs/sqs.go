package sqs

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
)

// NewFromConfig creates a new SQS client from aws.Config with OpenTelemetry instrumentation enabled.
func NewFromConfig(cfg aws.Config) *sqs.Client {
	// TODO: Is this correct without using the return values?
	otelaws.AppendMiddlewares(&cfg.APIOptions)
	return sqs.NewFromConfig(cfg)
}
