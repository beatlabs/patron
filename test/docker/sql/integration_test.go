// +build integration

package sql

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(RunWithSQL(m, 120*time.Second))
}

func TestRunTeardown(t *testing.T) {
	assert.Equal(t, "patron:test123@(localhost:3309)/patrondb?parseTime=true", DSN())
}
