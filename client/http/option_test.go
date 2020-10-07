package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransport(t *testing.T) {
	transport := &http.Transport{}
	client, err := New(Transport(transport))

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, transport, client.cl.Transport)
}
