package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_PprofHandlers(t *testing.T) {
	mux := http.NewServeMux()

	rr := profilingRoutes()

	for _, r := range rr {
		route, err := r.Build()
		assert.NoError(t, err)
		mux.HandleFunc(route.path, route.handler)
	}

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name string
		path string
		want int
	}{
		{"index", "/debug/pprof/", 200},
		{"cmdline", "/debug/pprof/allocs/", 200},
		{"cmdline", "/debug/pprof/cmdline/", 200},
		{"profile", "/debug/pprof/profile/?seconds=1", 200},
		{"symbol", "/debug/pprof/symbol/", 200},
		{"trace", "/debug/pprof/trace/?seconds=0.1", 200},
		{"heap", "/debug/pprof/heap/", 200},
		{"goroutine", "/debug/pprof/goroutine/", 200},
		{"block", "/debug/pprof/block/", 200},
		{"threadcreate", "/debug/pprof/threadcreate/", 200},
		{"mutex", "/debug/pprof/mutex/", 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("%s/%s", server.URL, tt.path))
			assert.NoError(t, err)
			assert.Equal(t, tt.want, resp.StatusCode)
		})
	}
}
