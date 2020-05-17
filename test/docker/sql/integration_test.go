// +build integration

package sql

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunTeardown(t *testing.T) {
	d, err := Create(120 * time.Second)
	assert.NoError(t, err)
	dsn := fmt.Sprintf("patron:test123@(localhost:%s)/patrondb?parseTime=true", d.Port())
	assert.Equal(t, dsn, d.DSN())
	assert.Len(t, d.Resources(), 1)
	assert.Empty(t, d.Teardown())
}
