package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_metricRoute(t *testing.T) {
	route := metricRoute()
	assert.Equal(t, http.MethodGet, route.method)
	assert.Equal(t, "/metrics", route.path)
	assert.NotNil(t, route.handler)
}
