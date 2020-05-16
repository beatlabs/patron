// +build integration

package sql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunTeardown(t *testing.T) {
	d, err := setup(120 * time.Second)
	assert.NoError(t, err)
	assert.Len(t, d.Resources(), 1)
	assert.Empty(t, d.Teardown())
}
