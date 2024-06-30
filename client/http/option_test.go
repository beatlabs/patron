package http

import (
	"net/http"
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func TestTransport(t *testing.T) {
	transport := &http.Transport{}
	client, err := New(WithTransport(transport))

	otelhttp.NewTransport(transport)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestTransport_Nil(t *testing.T) {
	client, err := New(WithTransport(nil))

	assert.Nil(t, client)
	require.Error(t, err, "transport must be supplied")
}

func TestCheckRedirect_Nil(t *testing.T) {
	client, err := New(WithCheckRedirect(nil))

	assert.Nil(t, client)
	require.Error(t, err, "check redirect must be supplied")
}

func TestCheckRedirect(t *testing.T) {
	cr := func(_ *http.Request, _ []*http.Request) error {
		return nil
	}

	client, err := New(WithCheckRedirect(cr))
	require.NoError(t, err)
	assert.NotNil(t, client)

	expFuncName := runtime.FuncForPC(reflect.ValueOf(cr).Pointer()).Name()
	actFuncName := runtime.FuncForPC(reflect.ValueOf(client.cl.CheckRedirect).Pointer()).Name()
	assert.Equal(t, expFuncName, actFuncName)
}
