package simple

import (
	"os"
	"testing"

	"github.com/beatlabs/patron/component/async/kafka/docker"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(docker.RunKafkaTest(m))
}

func TestInt(t *testing.T) {
	assert.Equal(t, 1, 1)
}
