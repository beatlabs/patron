package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupMetrics(t *testing.T) {
	deferFunc, read := SetupMetrics(context.Background(), t)
	defer deferFunc()
	assert.NotNil(t, deferFunc)
	assert.NotNil(t, read)
}
