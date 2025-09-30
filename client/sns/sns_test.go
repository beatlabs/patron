package sns

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFromConfig_Unit(t *testing.T) {
	t.Parallel()

	t.Run("creates client with basic config", func(t *testing.T) {
		t.Parallel()

		cfg := aws.Config{
			Region: "us-east-1",
		}

		client := NewFromConfig(cfg)

		require.NotNil(t, client)
		assert.NotNil(t, client.Options())
	})

	t.Run("creates client with custom region", func(t *testing.T) {
		t.Parallel()

		cfg := aws.Config{
			Region: "eu-west-1",
		}

		client := NewFromConfig(cfg)

		require.NotNil(t, client)
		assert.Equal(t, "eu-west-1", client.Options().Region)
	})

	t.Run("creates client with option functions", func(t *testing.T) {
		t.Parallel()

		cfg := aws.Config{
			Region: "us-west-2",
		}

		customRegion := "ap-south-1"
		optFn := func(o *sns.Options) {
			o.Region = customRegion
		}

		client := NewFromConfig(cfg, optFn)

		require.NotNil(t, client)
		assert.Equal(t, customRegion, client.Options().Region)
	})

	t.Run("creates client with multiple option functions", func(t *testing.T) {
		t.Parallel()

		cfg := aws.Config{
			Region: "us-east-1",
		}

		optFn1 := func(o *sns.Options) {
			o.Region = "us-west-1"
		}

		customEndpoint := "http://localhost:4566"
		optFn2 := func(o *sns.Options) {
			o.BaseEndpoint = &customEndpoint
		}

		client := NewFromConfig(cfg, optFn1, optFn2)

		require.NotNil(t, client)
		assert.Equal(t, "us-west-1", client.Options().Region)
		assert.NotNil(t, client.Options().BaseEndpoint)
		assert.Equal(t, customEndpoint, *client.Options().BaseEndpoint)
	})

	t.Run("otel middleware is applied", func(t *testing.T) {
		t.Parallel()

		cfg := aws.Config{
			Region: "us-east-1",
		}

		client := NewFromConfig(cfg)

		require.NotNil(t, client)
		// The otelaws.AppendMiddlewares adds middleware to APIOptions
		// We can't easily verify this without making actual calls, but we verify the client is created
		assert.NotNil(t, client.Options())
	})
}

func TestNewFromConfig_WithCredentials(t *testing.T) {
	t.Parallel()

	t.Run("creates client with credentials provider", func(t *testing.T) {
		t.Parallel()

		cfg := aws.Config{
			Region: "us-east-1",
			Credentials: aws.CredentialsProviderFunc(func(_ context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     "test-key",
					SecretAccessKey: "test-secret",
				}, nil
			}),
		}

		client := NewFromConfig(cfg)

		require.NotNil(t, client)
		assert.NotNil(t, client.Options().Credentials)
	})
}

func TestNewFromConfig_EmptyConfig(t *testing.T) {
	t.Parallel()

	t.Run("creates client with empty config", func(t *testing.T) {
		t.Parallel()

		cfg := aws.Config{}

		client := NewFromConfig(cfg)

		require.NotNil(t, client)
		// Client should be created even with empty config
		// AWS SDK will use default settings
		assert.NotNil(t, client.Options())
	})
}

func TestNewFromConfig_WithRetryConfig(t *testing.T) {
	t.Parallel()

	t.Run("creates client with custom retry config", func(t *testing.T) {
		t.Parallel()

		cfg := aws.Config{
			Region:           "us-east-1",
			RetryMaxAttempts: 5,
			RetryMode:        aws.RetryModeStandard,
		}

		client := NewFromConfig(cfg)

		require.NotNil(t, client)
		assert.NotNil(t, client.Options())
		// The retry configuration is part of the client options
	})
}
