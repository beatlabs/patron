package http

import (
	"net/http"
	"testing"

	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/stretchr/testify/assert"
)

func TestTransport(t *testing.T) {
	transport := &http.Transport{}
	client, err := New(Transport(transport))

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, &nethttp.Transport{RoundTripper: transport}, client.cl.Transport)
}

func TestTransport_Nil(t *testing.T) {
	client, err := New(Transport(nil))

	assert.Nil(t, client)
	assert.Error(t, err, "transport must be supplied")
}
