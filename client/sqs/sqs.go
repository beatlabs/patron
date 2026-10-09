package sqs

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	otelaws "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws" //nolint:staticcheck // SA1019: upstream deprecation being reverted, see open-telemetry/opentelemetry-go-contrib#9873
)

// NewFromConfig creates a new SQS client from aws.Config with OpenTelemetry instrumentation enabled.
func NewFromConfig(cfg aws.Config, optFns ...func(*sqs.Options)) *sqs.Client {
	otelaws.AppendMiddlewares(&cfg.APIOptions)
	return sqs.NewFromConfig(cfg, optFns...)
}
