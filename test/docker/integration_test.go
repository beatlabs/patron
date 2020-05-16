// +build integration

package docker

import (
	"testing"
	"time"

	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunTeardown(t *testing.T) {
	d, err := NewRuntime(10 * time.Second)
	require.NoError(t, err)

	runOptions := &dockertest.RunOptions{
		Repository:   "mysql",
		Tag:          "5.7.25",
		ExposedPorts: []string{"3306/tcp", "33060/tcp"},
	}
	mysql, err := d.RunWithOptions(runOptions)
	assert.NoError(t, err)
	assert.NotNil(t, mysql)
	ee := d.Teardown()
	assert.Empty(t, ee)
}
