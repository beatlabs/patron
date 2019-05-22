package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockAuthenticator struct {
	success bool
	err     error
}

func (mo MockAuthenticator) Authenticate(req *http.Request) (bool, error) {
	if mo.err != nil {
		return false, mo.err
	}
	return mo.success, nil
}

func TestNewRoute(t *testing.T) {
	r := NewRoute("/index", http.MethodGet, nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodGet, r.Method)
	assert.True(t, r.Trace)
	assert.NotNil(t, r.Auth)
}

func TestNewGetRoute(t *testing.T) {
	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	r := NewGetRoute("/index", nil, true, t1, t2)
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodGet, r.Method)
	assert.True(t, r.Trace)
	assert.Nil(t, r.Auth)
	assert.Len(t, r.Middlewares, 3)
}

func TestNewPostRoute(t *testing.T) {
	r := NewPostRoute("/index", nil, true)
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodPost, r.Method)
	assert.True(t, r.Trace)
	assert.Nil(t, r.Auth)
	assert.Len(t, r.Middlewares, 1)
}

func TestNewPutRoute(t *testing.T) {
	r := NewPutRoute("/index", nil, true)
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodPut, r.Method)
	assert.True(t, r.Trace)
	assert.Nil(t, r.Auth)
	assert.Len(t, r.Middlewares, 1)
}

func TestNewDeleteRoute(t *testing.T) {
	r := NewDeleteRoute("/index", nil, true)
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodDelete, r.Method)
	assert.True(t, r.Trace)
	assert.Nil(t, r.Auth)
	assert.Len(t, r.Middlewares, 1)
}

func TestNewPatchRoute(t *testing.T) {
	r := NewPatchRoute("/index", nil, true)
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodPatch, r.Method)
	assert.True(t, r.Trace)
	assert.Nil(t, r.Auth)
	assert.Len(t, r.Middlewares, 1)
}

func TestNewHeadRoute(t *testing.T) {
	r := NewHeadRoute("/index", nil, true)
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodHead, r.Method)
	assert.True(t, r.Trace)
	assert.Nil(t, r.Auth)
	assert.Len(t, r.Middlewares, 1)
}

func TestNewOptionsRoute(t *testing.T) {
	r := NewOptionsRoute("/index", nil, true)
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodOptions, r.Method)
	assert.True(t, r.Trace)
	assert.Nil(t, r.Auth)
	assert.Len(t, r.Middlewares, 1)
}
func TestNewRouteRaw(t *testing.T) {
	r := NewRouteRaw("/index", http.MethodGet, nil, false)
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, "GET", r.Method)
	assert.False(t, r.Trace)
	assert.Nil(t, r.Auth)
	assert.Len(t, r.Middlewares, 0)

	r = NewRouteRaw("/index", http.MethodGet, nil, true, tagMiddleware("t1"))
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, "GET", r.Method)
	assert.True(t, r.Trace)
	assert.Nil(t, r.Auth)
	assert.Len(t, r.Middlewares, 2)
}

func TestNewAuthGetRoute(t *testing.T) {
	r := NewAuthGetRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodGet, r.Method)
	assert.True(t, r.Trace)
	assert.NotNil(t, r.Auth)
	assert.Len(t, r.Middlewares, 2)
}

func TestNewAuthPostRoute(t *testing.T) {
	r := NewAuthPostRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodPost, r.Method)
	assert.True(t, r.Trace)
	assert.NotNil(t, r.Auth)
	assert.Len(t, r.Middlewares, 2)
}

func TestNewAuthPutRoute(t *testing.T) {
	r := NewAuthPutRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodPut, r.Method)
	assert.True(t, r.Trace)
	assert.NotNil(t, r.Auth)
	assert.Len(t, r.Middlewares, 2)
}

func TestNewAuthDeleteRoute(t *testing.T) {
	r := NewAuthDeleteRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodDelete, r.Method)
	assert.True(t, r.Trace)
	assert.NotNil(t, r.Auth)
	assert.Len(t, r.Middlewares, 2)
}

func TestNewAuthPatchRoute(t *testing.T) {
	r := NewAuthPatchRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodPatch, r.Method)
	assert.True(t, r.Trace)
	assert.NotNil(t, r.Auth)
	assert.Len(t, r.Middlewares, 2)
}

func TestNewAuthHeadRoute(t *testing.T) {
	r := NewAuthHeadRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodHead, r.Method)
	assert.True(t, r.Trace)
	assert.NotNil(t, r.Auth)
	assert.Len(t, r.Middlewares, 2)
}

func TestNewAuthOptionsRoute(t *testing.T) {
	r := NewAuthOptionsRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, http.MethodOptions, r.Method)
	assert.True(t, r.Trace)
	assert.NotNil(t, r.Auth)
	assert.Len(t, r.Middlewares, 2)
}

func TestNewAuthRouteRaw(t *testing.T) {
	r := NewAuthRouteRaw("/index", http.MethodGet, nil, false, &MockAuthenticator{})
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, "GET", r.Method)
	assert.False(t, r.Trace)
	assert.NotNil(t, r.Auth)
	assert.Len(t, r.Middlewares, 1)

	r = NewAuthRouteRaw("/index", http.MethodGet, nil, true, &MockAuthenticator{}, tagMiddleware("tag1"))
	assert.Equal(t, "/index", r.Pattern)
	assert.Equal(t, "GET", r.Method)
	assert.True(t, r.Trace)
	assert.NotNil(t, r.Auth)
	assert.Len(t, r.Middlewares, 3)
}
