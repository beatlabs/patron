package v2

import (
	"testing"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

func TestTimeout(t *testing.T) {
	cfg := amqp.Config{
		Locale: "123",
	}

	p := Publisher{}
	assert.NoError(t, WithConfig(cfg)(&p))
	assert.Equal(t, cfg, *p.cfg)
}
