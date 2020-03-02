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
			rb := NewRawRouteBuilder(MethodGet, tt.fields.path, func(http.ResponseWriter, *http.Request) {})
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
			rb := NewRawRouteBuilder(MethodGet, tt.fields.path, func(http.ResponseWriter, *http.Request) {})
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
			rb := NewRawRouteBuilder(MethodGet, tt.fields.path, func(http.ResponseWriter, *http.Request) {})
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

func TestRouteBuilder_Build(t *testing.T) {
	mockAuth := &MockAuthenticator{}
	mockProcessor := func(context.Context, *sync.Request) (*sync.Response, error) { return nil, nil }
	middleware := func(next http.Handler) http.Handler { return next }
	type fields struct {
		path string
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success":           {fields: fields{path: "/"}},
		"missing processor": {fields: fields{path: ""}, expectedErr: "path is empty\n"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewRouteBuilder(MethodGet, tt.fields.path, mockProcessor).WithTrace().WithAuth(mockAuth).
				WithMiddlewares(middleware).Build()

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Equal(t, Route{}, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestNewRawRouteBuilder(t *testing.T) {
	mockHandler := func(http.ResponseWriter, *http.Request) {}
	type args struct {
		method  Method
		path    string
		handler http.HandlerFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":         {args: args{method: MethodGet, path: "/", handler: mockHandler}},
		"invalid method":  {args: args{method: "", path: "/", handler: mockHandler}, expectedErr: "method is empty"},
		"invalid path":    {args: args{method: MethodGet, path: "", handler: mockHandler}, expectedErr: "path is empty"},
		"invalid handler": {args: args{method: MethodGet, path: "/", handler: nil}, expectedErr: "handler is nil"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRawRouteBuilder(tt.args.method, tt.args.path, tt.args.handler)

			if tt.expectedErr != "" {
				assert.Len(t, rb.errors, 1)
				assert.EqualError(t, rb.errors[0], tt.expectedErr)
			} else {
				assert.Len(t, rb.errors, 0)
			}
		})
	}
}

func TestNewRouteBuilder(t *testing.T) {
	mockProcessor := func(context.Context, *sync.Request) (*sync.Response, error) { return nil, nil }
	type args struct {
		method    Method
		path      string
		processor sync.ProcessorFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":         {args: args{method: MethodGet, path: "/", processor: mockProcessor}},
		"invalid method":  {args: args{method: "", path: "/", processor: mockProcessor}, expectedErr: "method is empty"},
		"invalid path":    {args: args{method: MethodGet, path: "", processor: mockProcessor}, expectedErr: "path is empty"},
		"invalid handler": {args: args{method: MethodGet, path: "/", processor: nil}, expectedErr: "processor is nil"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRouteBuilder(tt.args.method, tt.args.path, tt.args.processor)

			if tt.expectedErr != "" {
				assert.Len(t, rb.errors, 1)
				assert.EqualError(t, rb.errors[0], tt.expectedErr)
			} else {
				assert.Len(t, rb.errors, 0)
			}
		})
	}
}
