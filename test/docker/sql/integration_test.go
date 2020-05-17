// +build integration

package sql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunTeardown(t *testing.T) {
	d, err := create(120 * time.Second)
	assert.NoError(t, err)
	assert.NoError(t, d.setup())
	assert.Len(t, d.Resources(), 1)
	assert.Empty(t, d.Teardown())
}
