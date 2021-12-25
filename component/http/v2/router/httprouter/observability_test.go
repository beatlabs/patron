package httprouter

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
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

func Test_profilingRoutes(t *testing.T) {
	mux := httprouter.New()

	for _, route := range profilingRoutes() {
		mux.HandlerFunc(route.method, route.path, route.handler)
	}

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := map[string]struct {
		path string
		want int
	}{
		"index":        {"/debug/pprof/", 200},
		"allocs":       {"/debug/pprof/allocs/", 200},
		"cmdline":      {"/debug/pprof/cmdline/", 200},
		"profile":      {"/debug/pprof/profile/?seconds=1", 200},
		"symbol":       {"/debug/pprof/symbol/", 200},
		"trace":        {"/debug/pprof/trace/?seconds=1", 200},
		"heap":         {"/debug/pprof/heap/", 200},
		"goroutine":    {"/debug/pprof/goroutine/", 200},
		"block":        {"/debug/pprof/block/", 200},
		"threadcreate": {"/debug/pprof/threadcreate/", 200},
		"mutex":        {"/debug/pprof/mutex/", 200},
		"vars":         {"/debug/vars/", 200},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/%s", server.URL, tt.path))
			assert.NoError(t, err)
			assert.Equal(t, tt.want, resp.StatusCode)
		})
	}
}
