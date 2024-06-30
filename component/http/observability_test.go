package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type profilingTestCase struct {
	path string
	want int
}

func TestProfilingRoutes(t *testing.T) {
	t.Run("without vars", func(t *testing.T) {
		server := createProfilingServer(false)
		defer server.Close()

		for name, tt := range createProfilingTestCases(false) {
			tt := tt
			t.Run(name, func(t *testing.T) {
				req, err := http.NewRequestWithContext(context.Background(), "GET", fmt.Sprintf("%s/%s", server.URL, tt.path), nil)
				require.NoError(t, err)
				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				assert.Equal(t, tt.want, resp.StatusCode)
				require.NoError(t, resp.Body.Close())
			})
		}
	})

	t.Run("with vars", func(t *testing.T) {
		server := createProfilingServer(true)
		defer server.Close()

		for name, tt := range createProfilingTestCases(true) {
			tt := tt
			t.Run(name, func(t *testing.T) {
				req, err := http.NewRequestWithContext(context.Background(), "GET", fmt.Sprintf("%s/%s", server.URL, tt.path), nil)
				require.NoError(t, err)
				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				assert.Equal(t, tt.want, resp.StatusCode)
				require.NoError(t, resp.Body.Close())
			})
		}
	})
}

func createProfilingServer(enableExpVar bool) *httptest.Server {
	mux := http.NewServeMux()
	for _, route := range ProfilingRoutes(enableExpVar) {
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
