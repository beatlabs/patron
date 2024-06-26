package amqp

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeout(t *testing.T) {
	cfg := amqp.Config{
		Locale: "123",
	}

	p := Publisher{}
	require.NoError(t, WithConfig(cfg)(&p))
	assert.Equal(t, cfg, *p.cfg)
}
