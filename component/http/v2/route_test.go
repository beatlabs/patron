package v2

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRoute(t *testing.T) {
	t.Parallel()

	handler := func(http.ResponseWriter, *http.Request) {}

	type args struct {
		method      string
		path        string
		handler     http.HandlerFunc
		optionFuncs []RouteOptionFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{
			method:      http.MethodGet,
			path:        "/api",
			handler:     handler,
			optionFuncs: []RouteOptionFunc{RateLimiting(1, 1)},
		}},
		"missing method": {args: args{
			method:      "",
			path:        "/api",
			handler:     handler,
			optionFuncs: []RouteOptionFunc{RateLimiting(1, 1)},
		}, expectedErr: "method is empty"},
		"missing path": {args: args{
			method:      http.MethodGet,
			path:        "",
			handler:     handler,
			optionFuncs: []RouteOptionFunc{RateLimiting(1, 1)},
		}, expectedErr: "path is empty"},
		"missing handler": {args: args{
			method:      http.MethodGet,
			path:        "/api",
			handler:     nil,
			optionFuncs: []RouteOptionFunc{RateLimiting(1, 1)},
		}, expectedErr: "handler is nil"},
		"missing middlewares": {args: args{
			method:      http.MethodGet,
			path:        "/api",
			handler:     handler,
			optionFuncs: []RouteOptionFunc{Middlewares()},
		}, expectedErr: "middlewares are empty"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := NewRoute(tt.args.method, tt.args.path, tt.args.handler, tt.args.optionFuncs...)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.args.method, got.Method())
				assert.Equal(t, tt.args.path, got.Path())
				assert.NotNil(t, got.Handler())
				assert.Len(t, got.Middlewares(), 2)
				assert.Equal(t, "GET /api", got.String())
			}
		})
	}
}
