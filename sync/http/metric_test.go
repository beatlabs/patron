package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_metricRoute(t *testing.T) {
	route := metricRoute()
	assert.Equal(t, http.MethodGet, route.Method)
	assert.Equal(t, "/metrics", route.Path)
	assert.NotNil(t, route.Handler)
}
