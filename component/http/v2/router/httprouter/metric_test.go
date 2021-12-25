package httprouter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_metricRoute(t *testing.T) {
	route := metricRoute()
	assert.Equal(t, http.MethodGet, route.method)
	assert.Equal(t, "/metrics", route.path)

	resp := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	require.NoError(t, err)

	route.handler(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}
