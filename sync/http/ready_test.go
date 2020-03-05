package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_readyCheckRoute(t *testing.T) {
	tests := []struct {
		name string
		rcf  ReadyCheckFunc
		want int
	}{
		{"ready", func() ReadyStatus { return Ready }, http.StatusOK},
		{"notReady", func() ReadyStatus { return NotReady }, http.StatusServiceUnavailable},
		{"default", func() ReadyStatus { return 10 }, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := readyCheckRoute(tt.rcf)
			resp := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/alive", nil)
			assert.NoError(t, err)
			r.Handler(resp, req)
			assert.Equal(t, tt.want, resp.Code)
		})
	}
}
