package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupMetrics(t *testing.T) {
	shutdownFunc, collectMetrics := SetupMetrics(context.Background(), t)
	assert.NotNil(t, shutdownFunc)
	assert.NotNil(t, collectMetrics)
	shutdownFunc()
}
