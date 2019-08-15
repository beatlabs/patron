package sns

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Publisher_publishOpName(t *testing.T) {
	component := "component"
	p := &TracedPublisher{
		component: component,
	}

	msg, err := NewMessageBuilder().Build()
	require.NoError(t, err)

	assert.Equal(t, "component publish:unknown", p.publishOpName(*msg))
}
