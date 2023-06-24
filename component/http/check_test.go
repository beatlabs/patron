package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_aliveCheckRoute(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		acf  LivenessCheckFunc
		want int
	}{
		"alive":        {func() AliveStatus { return Alive }, http.StatusOK},
		"unresponsive": {func() AliveStatus { return Unhealthy }, http.StatusServiceUnavailable},
		"default":      {func() AliveStatus { return 10 }, http.StatusServiceUnavailable},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			route, err := LivenessCheckRoute(tt.acf)
			assert.NoError(t, err)
			assert.Equal(t, http.MethodGet, route.method)
			assert.Equal(t, "/alive", route.path)

			resp := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/alive", nil)
			require.NoError(t, err)

			route.handler(resp, req)

			assert.Equal(t, tt.want, resp.Code)
		})
	}
}

func Test_readyCheckRoute(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		rcf  ReadyCheckFunc
		want int
	}{
		"ready":    {func() ReadyStatus { return Ready }, http.StatusOK},
		"notReady": {func() ReadyStatus { return NotReady }, http.StatusServiceUnavailable},
		"default":  {func() ReadyStatus { return 10 }, http.StatusServiceUnavailable},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			route, err := ReadyCheckRoute(tt.rcf)
			assert.NoError(t, err)
			assert.Equal(t, http.MethodGet, route.method)
			assert.Equal(t, "/ready", route.path)

			resp := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/ready", nil)
			require.NoError(t, err)

			route.handler(resp, req)

			assert.Equal(t, tt.want, resp.Code)
		})
	}
}
