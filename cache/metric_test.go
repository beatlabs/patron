package cache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestUseCaseAttribute(t *testing.T) {
	assert.Equal(t, attribute.String("cache.use_case", "test"), UseCaseAttribute("test"))
}

func TestSetupAndUseMetrics(t *testing.T) {
	SetupMetricsOnce()

	assert.NotNil(t, cacheCounter)

	ObserveHit(context.Background(), attribute.String("test", "test"))

	ObserveMiss(context.Background(), attribute.String("test", "test"))
}
