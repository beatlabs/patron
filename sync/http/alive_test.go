package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_aliveCheckRoute(t *testing.T) {
	tests := []struct {
		name string
		acf  AliveCheckFunc
		want int
	}{
		{"alive", func() AliveStatus { return Alive }, http.StatusOK},
		{"unresponsive", func() AliveStatus { return Unresponsive }, http.StatusServiceUnavailable},
		{"default", func() AliveStatus { return 10 }, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := aliveCheckRoute(tt.acf).Build()
			assert.NoError(t, err)
			resp := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "/alive", nil)
			assert.NoError(t, err)
			r.handler(resp, req)
			assert.Equal(t, tt.want, resp.Code)
		})
	}
}
