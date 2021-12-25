package httprouter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_readyCheckRoute(t *testing.T) {
	tests := map[string]struct {
		rcf  ReadyCheckFunc
		want int
	}{
		"ready":    {func() ReadyStatus { return Ready }, http.StatusOK},
		"notReady": {func() ReadyStatus { return NotReady }, http.StatusServiceUnavailable},
		"default":  {func() ReadyStatus { return 10 }, http.StatusOK},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			route := readyCheckRoute(tt.rcf)
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
