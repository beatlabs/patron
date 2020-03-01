package http

import (
	"context"
	"net/http"
	"testing"

	"github.com/beatlabs/patron/sync"
	"github.com/beatlabs/patron/sync/http/auth"
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
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodGet, r.method)
}

func TestNewGetRoute(t *testing.T) {
	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	r := NewGetRoute("/index", nil, true, t1, t2)
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodGet, r.method)
	assert.Len(t, r.middlewares, 3)
}

func TestNewPostRoute(t *testing.T) {
	r := NewPostRoute("/index", nil, true)
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodPost, r.method)
	assert.Len(t, r.middlewares, 1)
}

func TestNewPutRoute(t *testing.T) {
	r := NewPutRoute("/index", nil, true)
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodPut, r.method)
	assert.Len(t, r.middlewares, 1)
}

func TestNewDeleteRoute(t *testing.T) {
	r := NewDeleteRoute("/index", nil, true)
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodDelete, r.method)
	assert.Len(t, r.middlewares, 1)
}

func TestNewPatchRoute(t *testing.T) {
	r := NewPatchRoute("/index", nil, true)
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodPatch, r.method)
	assert.Len(t, r.middlewares, 1)
}

func TestNewHeadRoute(t *testing.T) {
	r := NewHeadRoute("/index", nil, true)
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodHead, r.method)
	assert.Len(t, r.middlewares, 1)
}

func TestNewOptionsRoute(t *testing.T) {
	r := NewOptionsRoute("/index", nil, true)
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodOptions, r.method)
	assert.Len(t, r.middlewares, 1)
}
func TestNewRouteRaw(t *testing.T) {
	r := NewRouteRaw("/index", http.MethodGet, nil, false)
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, "GET", r.method)
	assert.Len(t, r.middlewares, 0)

	r = NewRouteRaw("/index", http.MethodGet, nil, true, tagMiddleware("t1"))
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, "GET", r.method)
	assert.Len(t, r.middlewares, 2)
}

func TestNewAuthGetRoute(t *testing.T) {
	r := NewAuthGetRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodGet, r.method)
	assert.Len(t, r.middlewares, 2)
}

func TestNewAuthPostRoute(t *testing.T) {
	r := NewAuthPostRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodPost, r.method)
	assert.Len(t, r.middlewares, 2)
}

func TestNewAuthPutRoute(t *testing.T) {
	r := NewAuthPutRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodPut, r.method)
	assert.Len(t, r.middlewares, 2)
}

func TestNewAuthDeleteRoute(t *testing.T) {
	r := NewAuthDeleteRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodDelete, r.method)
	assert.Len(t, r.middlewares, 2)
}

func TestNewAuthPatchRoute(t *testing.T) {
	r := NewAuthPatchRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodPatch, r.method)
	assert.Len(t, r.middlewares, 2)
}

func TestNewAuthHeadRoute(t *testing.T) {
	r := NewAuthHeadRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodHead, r.method)
	assert.Len(t, r.middlewares, 2)
}

func TestNewAuthOptionsRoute(t *testing.T) {
	r := NewAuthOptionsRoute("/index", nil, true, &MockAuthenticator{})
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, http.MethodOptions, r.method)
	assert.Len(t, r.middlewares, 2)
}

func TestNewAuthRouteRaw(t *testing.T) {
	r := NewAuthRouteRaw("/index", http.MethodGet, nil, false, &MockAuthenticator{})
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, "GET", r.method)
	assert.Len(t, r.middlewares, 1)

	r = NewAuthRouteRaw("/index", http.MethodGet, nil, true, &MockAuthenticator{}, tagMiddleware("tag1"))
	assert.Equal(t, "/index", r.path)
	assert.Equal(t, "GET", r.method)
	assert.Len(t, r.middlewares, 3)
}

func TestRouteBuilder_WithTrace(t *testing.T) {
	type fields struct {
		path string
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success":      {fields: fields{path: "/"}},
		"missing path": {fields: fields{path: ""}, expectedErr: "path is empty"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRouteBuilder(MethodGet, tt.fields.path)
			rb.WithTrace()

			if tt.expectedErr != "" {
				assert.Len(t, rb.errors, 1)
				assert.EqualError(t, rb.errors[0], tt.expectedErr)
			} else {
				assert.Len(t, rb.errors, 0)
				assert.True(t, rb.trace)
			}
		})
	}
}

func TestRouteBuilder_WithMiddlewares(t *testing.T) {
	middleware := func(next http.Handler) http.Handler { return next }
	type fields struct {
		path        string
		middlewares []MiddlewareFunc
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success":            {fields: fields{path: "/", middlewares: []MiddlewareFunc{middleware}}},
		"missing path":       {fields: fields{path: ""}, expectedErr: "path is empty"},
		"missing middleware": {fields: fields{path: "/"}, expectedErr: "middlewares are empty"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRouteBuilder(MethodGet, tt.fields.path)
			if len(tt.fields.middlewares) == 0 {
				rb.WithMiddlewares()
			} else {
				rb.WithMiddlewares(tt.fields.middlewares...)
			}

			if tt.expectedErr != "" {
				assert.Len(t, rb.errors, 1)
				assert.EqualError(t, rb.errors[0], tt.expectedErr)
			} else {
				assert.Len(t, rb.errors, 0)
				assert.Len(t, rb.middlewares, 1)
			}
		})
	}
}

func TestRouteBuilder_WithAuth(t *testing.T) {
	mockAuth := &MockAuthenticator{}
	type fields struct {
		path          string
		authenticator auth.Authenticator
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success":            {fields: fields{path: "/", authenticator: mockAuth}},
		"missing path":       {fields: fields{path: ""}, expectedErr: "path is empty"},
		"missing middleware": {fields: fields{path: "/"}, expectedErr: "authenticator is nil"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRouteBuilder(MethodGet, tt.fields.path)
			rb.WithAuth(tt.fields.authenticator)

			if tt.expectedErr != "" {
				assert.Len(t, rb.errors, 1)
				assert.EqualError(t, rb.errors[0], tt.expectedErr)
			} else {
				assert.Len(t, rb.errors, 0)
				assert.NotNil(t, rb.authenticator)
			}
		})
	}
}

func TestRouteBuilder_WithProcessor(t *testing.T) {
	mockProcessor := func(ctx context.Context, r *sync.Request) (*sync.Response, error) { return nil, nil }
	type fields struct {
		path      string
		processor sync.ProcessorFunc
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success":            {fields: fields{path: "/", processor: mockProcessor}},
		"missing path":       {fields: fields{path: ""}, expectedErr: "path is empty"},
		"missing middleware": {fields: fields{path: "/"}, expectedErr: "processor func is nil"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRouteBuilder(MethodGet, tt.fields.path)
			rb.WithProcessor(tt.fields.processor)

			if tt.expectedErr != "" {
				assert.Len(t, rb.errors, 1)
				assert.EqualError(t, rb.errors[0], tt.expectedErr)
			} else {
				assert.Len(t, rb.errors, 0)
				assert.NotNil(t, rb.handler)
			}
		})
	}
}

func TestRouteBuilder_WithRawHandler(t *testing.T) {
	mockHandler := func(http.ResponseWriter, *http.Request) {}
	type fields struct {
		path    string
		handler http.HandlerFunc
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success":            {fields: fields{path: "/", handler: mockHandler}},
		"missing path":       {fields: fields{path: ""}, expectedErr: "path is empty"},
		"missing middleware": {fields: fields{path: "/"}, expectedErr: "raw handler func is nil"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRouteBuilder(MethodGet, tt.fields.path)
			rb.WithRawHandler(tt.fields.handler)

			if tt.expectedErr != "" {
				assert.Len(t, rb.errors, 1)
				assert.EqualError(t, rb.errors[0], tt.expectedErr)
			} else {
				assert.Len(t, rb.errors, 0)
				assert.NotNil(t, rb.handler)
			}
		})
	}
}

func TestRouteBuilder_Build(t *testing.T) {
	mockHandler := func(http.ResponseWriter, *http.Request) {}
	type fields struct {
		path    string
		handler http.HandlerFunc
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success":            {fields: fields{path: "/", handler: mockHandler}},
		"missing path":       {fields: fields{path: ""}, expectedErr: "path is empty\n"},
		"missing middleware": {fields: fields{path: "/"}, expectedErr: "handler is nil"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRouteBuilder(MethodGet, tt.fields.path)
			if tt.fields.handler != nil {
				rb = rb.WithRawHandler(tt.fields.handler)
			}
			got, err := rb.Build()

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}
