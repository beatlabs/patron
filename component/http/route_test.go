package http

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRoute(t *testing.T) {
	t.Parallel()

	handler := func(http.ResponseWriter, *http.Request) {}
	rateLimiting, err := WithRateLimiting(1, 1)
	require.NoError(t, err)

	type args struct {
		path        string
		handler     http.HandlerFunc
		optionFuncs []RouteOptionFunc
	}

	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{
			path:        "GET /api",
			handler:     handler,
			optionFuncs: []RouteOptionFunc{rateLimiting},
		}},
		"missing path": {args: args{
			path:        "",
			handler:     handler,
			optionFuncs: []RouteOptionFunc{rateLimiting},
		}, expectedErr: "path is empty"},
		"missing handler": {args: args{
			path:        "GET /api",
			handler:     nil,
			optionFuncs: []RouteOptionFunc{rateLimiting},
		}, expectedErr: "handler is nil"},
		"missing middlewares": {args: args{
			path:        "GET /api",
			handler:     handler,
			optionFuncs: []RouteOptionFunc{WithMiddlewares()},
		}, expectedErr: "middlewares are empty"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := NewRoute(tt.args.path, tt.args.handler, tt.args.optionFuncs...)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assertRoute(t, tt.args.path, got)
				assert.Equal(t, "GET /api", got.String())
			}
		})
	}
}

func assertRoute(t *testing.T, path string, route *Route) {
	assert.Equal(t, path, route.Path())
	assert.NotNil(t, route.Handler())
	assert.Len(t, route.Middlewares(), 1)
}

func TestRoutes_Append(t *testing.T) {
	t.Parallel()
	type args struct {
		route *Route
		err   error
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":      {args: args{route: &Route{}, err: nil}},
		"error exist":  {args: args{route: &Route{}, err: errors.New("TEST")}, expectedErr: "TEST"},
		"route is nil": {args: args{route: nil, err: nil}, expectedErr: "route is nil"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := &Routes{}
			r.Append(tt.args.route, tt.args.err)
			routes, err := r.Result()
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				assert.Empty(t, routes)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, routes)
			}
		})
	}
}
