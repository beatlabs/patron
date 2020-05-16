package sql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDSN(t *testing.T) {
	assert.Equal(t, "patron:test123@(localhost:3309)/patrondb?parseTime=true", DSN())
}
