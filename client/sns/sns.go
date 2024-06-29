package sns

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
)

// NewFromConfig creates a new SNS client from aws.Config with OpenTelemetry instrumentation enabled.
func NewFromConfig(cfg aws.Config) *sns.Client {
	otelaws.AppendMiddlewares(&cfg.APIOptions)
	return sns.NewFromConfig(cfg)
}
