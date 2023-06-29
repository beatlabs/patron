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
			optionFuncs: []RouteOptionFunc{rateLimiting},
		}},
		"missing method": {args: args{
			method:      "",
			path:        "/api",
			handler:     handler,
			optionFuncs: []RouteOptionFunc{rateLimiting},
		}, expectedErr: "method is empty"},
		"missing path": {args: args{
			method:      http.MethodGet,
			path:        "",
			handler:     handler,
			optionFuncs: []RouteOptionFunc{rateLimiting},
		}, expectedErr: "path is empty"},
		"missing handler": {args: args{
			method:      http.MethodGet,
			path:        "/api",
			handler:     nil,
			optionFuncs: []RouteOptionFunc{rateLimiting},
		}, expectedErr: "handler is nil"},
		"missing middlewares": {args: args{
			method:      http.MethodGet,
			path:        "/api",
			handler:     handler,
			optionFuncs: []RouteOptionFunc{WithMiddlewares()},
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
				assertRoute(t, tt.args.method, tt.args.path, got)
				assert.Equal(t, "GET /api", got.String())
			}
		})
	}
}

func TestNewGetRoute(t *testing.T) {
	rateLimiting, err := WithRateLimiting(1, 1)
	require.NoError(t, err)
	route, err := NewGetRoute("/api", func(writer http.ResponseWriter, request *http.Request) {},
		[]RouteOptionFunc{rateLimiting}...)
	require.NoError(t, err)
	assertRoute(t, http.MethodGet, "/api", route)
}

func TestNewHeadRoute(t *testing.T) {
	rateLimiting, err := WithRateLimiting(1, 1)
	require.NoError(t, err)
	route, err := NewHeadRoute("/api", func(writer http.ResponseWriter, request *http.Request) {},
		[]RouteOptionFunc{rateLimiting}...)
	require.NoError(t, err)
	assertRoute(t, http.MethodHead, "/api", route)
}

func TestNewPostRoute(t *testing.T) {
	rateLimiting, err := WithRateLimiting(1, 1)
	require.NoError(t, err)
	route, err := NewPostRoute("/api", func(writer http.ResponseWriter, request *http.Request) {},
		[]RouteOptionFunc{rateLimiting}...)
	require.NoError(t, err)
	assertRoute(t, http.MethodPost, "/api", route)
}

func TestNewPutRoute(t *testing.T) {
	rateLimiting, err := WithRateLimiting(1, 1)
	require.NoError(t, err)
	route, err := NewPutRoute("/api", func(writer http.ResponseWriter, request *http.Request) {},
		[]RouteOptionFunc{rateLimiting}...)
	require.NoError(t, err)
	assertRoute(t, http.MethodPut, "/api", route)
}

func TestNewPatchRoute(t *testing.T) {
	rateLimiting, err := WithRateLimiting(1, 1)
	require.NoError(t, err)
	route, err := NewPatchRoute("/api", func(writer http.ResponseWriter, request *http.Request) {},
		[]RouteOptionFunc{rateLimiting}...)
	require.NoError(t, err)
	assertRoute(t, http.MethodPatch, "/api", route)
}

func TestNewDeleteRoute(t *testing.T) {
	rateLimiting, err := WithRateLimiting(1, 1)
	require.NoError(t, err)
	route, err := NewDeleteRoute("/api", func(writer http.ResponseWriter, request *http.Request) {},
		[]RouteOptionFunc{rateLimiting}...)
	require.NoError(t, err)
	assertRoute(t, http.MethodDelete, "/api", route)
}

func TestNewConnectRoute(t *testing.T) {
	rateLimiting, err := WithRateLimiting(1, 1)
	require.NoError(t, err)
	route, err := NewConnectRoute("/api", func(writer http.ResponseWriter, request *http.Request) {},
		[]RouteOptionFunc{rateLimiting}...)
	require.NoError(t, err)
	assertRoute(t, http.MethodConnect, "/api", route)
}

func TestNewOptionsRoute(t *testing.T) {
	rateLimiting, err := WithRateLimiting(1, 1)
	require.NoError(t, err)
	route, err := NewOptionsRoute("/api", func(writer http.ResponseWriter, request *http.Request) {},
		[]RouteOptionFunc{rateLimiting}...)
	require.NoError(t, err)
	assertRoute(t, http.MethodOptions, "/api", route)
}

func TestNewTraceRoute(t *testing.T) {
	rateLimiting, err := WithRateLimiting(1, 1)
	require.NoError(t, err)
	route, err := NewTraceRoute("/api", func(writer http.ResponseWriter, request *http.Request) {},
		[]RouteOptionFunc{rateLimiting}...)
	require.NoError(t, err)
	assertRoute(t, http.MethodTrace, "/api", route)
}

func assertRoute(t *testing.T, method, path string, route *Route) {
	assert.Equal(t, method, route.Method())
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
		"error exist":  {args: args{route: &Route{}, err: errors.New("TEST")}, expectedErr: "TEST\n"},
		"route is nil": {args: args{route: nil, err: nil}, expectedErr: "route is nil\n"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := &Routes{}
			r.Append(tt.args.route, tt.args.err)
			routes, err := r.Result()
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Empty(t, routes)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, routes)
			}
		})
	}
}
