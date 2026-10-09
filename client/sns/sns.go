package sns

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	otelaws "go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws" //nolint:staticcheck // SA1019: upstream deprecation being reverted, see open-telemetry/opentelemetry-go-contrib#9873
)

// NewFromConfig creates a new SNS client from aws.Config with OpenTelemetry instrumentation enabled.
func NewFromConfig(cfg aws.Config, optFns ...func(*sns.Options)) *sns.Client {
	otelaws.AppendMiddlewares(&cfg.APIOptions)
	return sns.NewFromConfig(cfg, optFns...)
}
