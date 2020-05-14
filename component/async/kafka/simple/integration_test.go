// +build integration

package simple

import (
	"os"
	"testing"
	"time"

	"github.com/beatlabs/patron/component/async/kafka/test"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	os.Exit(test.RunWithKafka(m, 120*time.Second, "Topic1:1:1"))
}

func TestInt(t *testing.T) {
	assert.Equal(t, 1, 1)
}
