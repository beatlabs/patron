package http

import (
	"net/http"
	"reflect"
	"runtime"
	"testing"

	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/stretchr/testify/assert"
)

func TestTransport(t *testing.T) {
	transport := &http.Transport{}
	client, err := New(WithTransport(transport))

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, &nethttp.Transport{RoundTripper: transport}, client.cl.Transport)
}

func TestTransport_Nil(t *testing.T) {
	client, err := New(WithTransport(nil))

	assert.Nil(t, client)
	assert.Error(t, err, "transport must be supplied")
}

func TestCheckRedirect_Nil(t *testing.T) {
	client, err := New(WithCheckRedirect(nil))

	assert.Nil(t, client)
	assert.Error(t, err, "check redirect must be supplied")
}

func TestCheckRedirect(t *testing.T) {
	cr := func(req *http.Request, via []*http.Request) error {
		return nil
	}

	client, err := New(WithCheckRedirect(cr))
	assert.NoError(t, err)
	assert.NotNil(t, client)

	expFuncName := runtime.FuncForPC(reflect.ValueOf(cr).Pointer()).Name()
	actFuncName := runtime.FuncForPC(reflect.ValueOf(client.cl.CheckRedirect).Pointer()).Name()
	assert.Equal(t, expFuncName, actFuncName)
}
