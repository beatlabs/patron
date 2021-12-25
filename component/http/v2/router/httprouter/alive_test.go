package httprouter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_aliveCheckRoute(t *testing.T) {
	tests := map[string]struct {
		acf  AliveCheckFunc
		want int
	}{
		"alive":        {func() AliveStatus { return Alive }, http.StatusOK},
		"unresponsive": {func() AliveStatus { return Unresponsive }, http.StatusServiceUnavailable},
		"default":      {func() AliveStatus { return 10 }, http.StatusOK},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			route := aliveCheckRoute(tt.acf)
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
