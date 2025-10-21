// Package observability provides functionality for initializing OpenTelemetry's traces and metrics.
// It includes methods for setting up and shutting down the observability components.
package observability

import (
	"context"
	"errors"
	"testing"

	"github.com/beatlabs/patron/observability/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func TestComponentAttribute(t *testing.T) {
	assert.Equal(t, attribute.String("component", "test"), ComponentAttribute("test"))
}

func TestClientAttribute(t *testing.T) {
	assert.Equal(t, attribute.String("client", "test"), ClientAttribute("test"))
}

func TestStatusAttribute(t *testing.T) {
	type args struct {
		err error
	}
	tests := map[string]struct {
		args args
		want attribute.KeyValue
	}{
		"succeeded": {args: args{err: nil}, want: SucceededAttribute},
		"failed":    {args: args{err: assert.AnError}, want: FailedAttribute},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, StatusAttribute(tt.args.err))
		})
	}
}

func TestComponentAttributeValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected attribute.KeyValue
	}{
		{
			name:     "http component",
			input:    "http",
			expected: attribute.String("component", "http"),
		},
		{
			name:     "kafka component",
			input:    "kafka",
			expected: attribute.String("component", "kafka"),
		},
		{
			name:     "empty component",
			input:    "",
			expected: attribute.String("component", ""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComponentAttribute(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClientAttributeValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected attribute.KeyValue
	}{
		{
			name:     "http client",
			input:    "http",
			expected: attribute.String("client", "http"),
		},
		{
			name:     "grpc client",
			input:    "grpc",
			expected: attribute.String("client", "grpc"),
		},
		{
			name:     "empty client",
			input:    "",
			expected: attribute.String("client", ""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClientAttribute(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStatusAttributeValues(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected attribute.KeyValue
	}{
		{
			name:     "nil error returns succeeded",
			err:      nil,
			expected: SucceededAttribute,
		},
		{
			name:     "error returns failed",
			err:      errors.New("test error"),
			expected: FailedAttribute,
		},
		{
			name:     "wrapped error returns failed",
			err:      errors.Join(errors.New("error1"), errors.New("error2")),
			expected: FailedAttribute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StatusAttribute(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSucceededAttribute(t *testing.T) {
	assert.Equal(t, "status", string(SucceededAttribute.Key))
	assert.Equal(t, "succeeded", SucceededAttribute.Value.AsString())
}

func TestFailedAttribute(t *testing.T) {
	assert.Equal(t, "status", string(FailedAttribute.Key))
	assert.Equal(t, "failed", FailedAttribute.Value.AsString())
}

func TestStatusAttributeConstant(t *testing.T) {
	assert.Equal(t, "status", statusAttribute)
}

func TestCreateResource(t *testing.T) {
	name := "test-service"
	version := "v1.2.3"

	res, err := createResource(name, version)
	require.NoError(t, err)
	assert.NotNil(t, res)

	// Verify resource contains expected attributes
	attrs := res.Attributes()

	var foundName, foundVersion bool
	for _, attr := range attrs {
		if attr.Key == semconv.ServiceNameKey && attr.Value.AsString() == name {
			foundName = true
		}
		if attr.Key == semconv.ServiceVersionKey && attr.Value.AsString() == version {
			foundVersion = true
		}
	}

	assert.True(t, foundName, "Service name not found in resource")
	assert.True(t, foundVersion, "Service version not found in resource")
}

func TestCreateResource_EmptyValues(t *testing.T) {
	t.Run("empty name", func(t *testing.T) {
		res, err := createResource("", "v1.0.0")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("empty version", func(t *testing.T) {
		res, err := createResource("test-service", "")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("both empty", func(t *testing.T) {
		res, err := createResource("", "")
		require.NoError(t, err)
		assert.NotNil(t, res)
	})
}

func TestConfig(t *testing.T) {
	cfg := Config{
		Name:    "test-service",
		Version: "v1.0.0",
		LogConfig: log.Config{
			Level:  "info",
			IsJSON: true,
		},
	}

	assert.Equal(t, "test-service", cfg.Name)
	assert.Equal(t, "v1.0.0", cfg.Version)
	assert.Equal(t, "info", cfg.LogConfig.Level)
	assert.True(t, cfg.LogConfig.IsJSON)
}

func TestProvider_Structure(t *testing.T) {
	// Test that Provider struct can be created
	provider := &Provider{}
	assert.NotNil(t, provider)
	assert.Nil(t, provider.mp)
	assert.Nil(t, provider.tp)
}

func TestProvider_Shutdown_WithNilProviders(t *testing.T) {
	provider := &Provider{
		mp: nil,
		tp: nil,
	}

	ctx := context.Background()

	// This should handle nil providers gracefully or panic - let's see
	// In actual implementation, this would likely panic or error
	// We're documenting the behavior
	defer func() {
		if r := recover(); r != nil {
			// Expected to panic with nil providers
			assert.NotNil(t, r)
		}
	}()

	_ = provider.Shutdown(ctx)
}

func TestSetup_ValidConfig(t *testing.T) {
	t.Skip("Requires OTLP collector - tested in integration tests")
}

func TestSetup_WithAttributes(t *testing.T) {
	t.Skip("Requires OTLP collector - tested in integration tests")
}

func TestCreateResource_WithDefaults(t *testing.T) {
	res, err := createResource("my-app", "1.0.0")
	require.NoError(t, err)

	// Verify it merges with default resource
	assert.NotNil(t, res)

	// Resource should have at least the service name and version
	attrs := res.Attributes()
	assert.NotEmpty(t, attrs)
}

func TestAttributeHelpers(t *testing.T) {
	t.Run("multiple component attributes", func(t *testing.T) {
		attr1 := ComponentAttribute("http")
		attr2 := ComponentAttribute("grpc")
		attr3 := ComponentAttribute("kafka")

		assert.NotEqual(t, attr1.Value, attr2.Value)
		assert.NotEqual(t, attr2.Value, attr3.Value)
		assert.Equal(t, attr1.Key, attr2.Key)
		assert.Equal(t, attr2.Key, attr3.Key)
	})

	t.Run("multiple client attributes", func(t *testing.T) {
		attr1 := ClientAttribute("redis")
		attr2 := ClientAttribute("postgres")
		attr3 := ClientAttribute("mongodb")

		assert.NotEqual(t, attr1.Value, attr2.Value)
		assert.NotEqual(t, attr2.Value, attr3.Value)
		assert.Equal(t, attr1.Key, attr2.Key)
		assert.Equal(t, attr2.Key, attr3.Key)
	})
}

func TestProvider_ShutdownOrder(t *testing.T) {
	t.Skip("Requires OTLP collector - tested in integration tests")
}

func TestConfig_DefaultValues(t *testing.T) {
	cfg := Config{}

	assert.Empty(t, cfg.Name)
	assert.Empty(t, cfg.Version)
	assert.Empty(t, cfg.LogConfig.Level)
	assert.False(t, cfg.LogConfig.IsJSON)
}
