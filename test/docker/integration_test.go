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
		Repository: "nats",
		Tag:        "2.1.7-scratch",
	}
	resource, err := d.RunWithOptions(runOptions)
	assert.NoError(t, err)
	assert.NotNil(t, resource)
	ee := d.Teardown()
	assert.Empty(t, ee)
}
