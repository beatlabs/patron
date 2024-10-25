package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/beatlabs/patron/observability/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type profilingTestCase struct {
	path string
	want int
}

func TestProfilingRoutes(t *testing.T) {
	t.Run("without vars", func(t *testing.T) {
		server := createServer(false)
		defer server.Close()

		for name, tt := range createProfilingTestCases(false) {
			t.Run(name, func(t *testing.T) {
				req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("%s/%s", server.URL, tt.path), nil)
				require.NoError(t, err)
				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				assert.Equal(t, tt.want, resp.StatusCode)
				require.NoError(t, resp.Body.Close())
			})
		}
	})

	t.Run("with vars", func(t *testing.T) {
		server := createServer(true)
		defer server.Close()

		for name, tt := range createProfilingTestCases(true) {
			t.Run(name, func(t *testing.T) {
				req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("%s/%s", server.URL, tt.path), nil)
				require.NoError(t, err)
				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				assert.Equal(t, tt.want, resp.StatusCode)
				require.NoError(t, resp.Body.Close())
			})
		}
	})
}

func createServer(enableExpVar bool) *httptest.Server {
	mux := http.NewServeMux()
	for _, route := range ProfilingRoutes(enableExpVar) {
		mux.HandleFunc(route.path, route.handler)
	}
	for _, route := range LoggingRoutes() {
		mux.HandleFunc(route.path, route.handler)
	}

	return httptest.NewServer(mux)
}

func createProfilingTestCases(enableExpVar bool) map[string]profilingTestCase {
	expVarWant := 404
	if enableExpVar {
		expVarWant = 200
	}

	return map[string]profilingTestCase{
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
		"vars":         {"/debug/vars/", expVarWant},
	}
}

func TestLoggingRoutes(t *testing.T) {
	log.Setup(&log.Config{
		IsJSON: true,
		Level:  "info",
	})
	server := createServer(true)
	defer server.Close()

	t.Run("change log level to debug", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, fmt.Sprintf("%s/debug/log/debug", server.URL), nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, resp.Body.Close())
	})

	t.Run("wrong log level", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, fmt.Sprintf("%s/debug/log/xxx", server.URL), nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		require.NoError(t, resp.Body.Close())
	})

	t.Run("empty log level", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, fmt.Sprintf("%s/debug/log/", server.URL), nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		require.NoError(t, resp.Body.Close())
	})
}
