package http

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/beatlabs/patron/component/http/auth"
	"github.com/beatlabs/patron/component/http/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockAuthenticator struct {
	success bool
	err     error
}

func (mo MockAuthenticator) Authenticate(_ *http.Request) (bool, error) {
	if mo.err != nil {
		return false, mo.err
	}
	return mo.success, nil
}

func TestRouteBuilder_WithMethodGet(t *testing.T) {
	type args struct {
		methodExists bool
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":               {args: args{}},
		"method already exists": {args: args{methodExists: true}, expectedErr: "method already set\n"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {})
			if tt.args.methodExists {
				rb.MethodGet()
			}
			got, err := rb.MethodGet().Build()

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Equal(t, Route{}, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, http.MethodGet, got.method)
			}
		})
	}
}

func TestRouteBuilder_WithMethodPost(t *testing.T) {
	rb := NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).MethodPost()
	assert.Equal(t, http.MethodPost, rb.method)
}

func TestRouteBuilder_WithMethodPut(t *testing.T) {
	rb := NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).MethodPut()
	assert.Equal(t, http.MethodPut, rb.method)
}

func TestRouteBuilder_WithMethodPatch(t *testing.T) {
	rb := NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).MethodPatch()
	assert.Equal(t, http.MethodPatch, rb.method)
}

func TestRouteBuilder_WithMethodConnect(t *testing.T) {
	rb := NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).MethodConnect()
	assert.Equal(t, http.MethodConnect, rb.method)
}

func TestRouteBuilder_WithMethodDelete(t *testing.T) {
	rb := NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).MethodDelete()
	assert.Equal(t, http.MethodDelete, rb.method)
}

func TestRouteBuilder_WithMethodHead(t *testing.T) {
	rb := NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).MethodHead()
	assert.Equal(t, http.MethodHead, rb.method)
}

func TestRouteBuilder_WithMethodTrace(t *testing.T) {
	rb := NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).MethodTrace()
	assert.Equal(t, http.MethodTrace, rb.method)
}

func TestRouteBuilder_WithMethodOptions(t *testing.T) {
	rb := NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).MethodOptions()
	assert.Equal(t, http.MethodOptions, rb.method)
}

func TestRouteBuilder_WithTrace(t *testing.T) {
	rb := NewRawRouteBuilder("/", func(http.ResponseWriter, *http.Request) {}).WithTrace()
	assert.True(t, rb.trace)
}

func TestRouteBuilder_WithMiddlewares(t *testing.T) {
	middleware := func(next http.Handler) http.Handler { return next }
	mockHandler := func(http.ResponseWriter, *http.Request) {}
	type fields struct {
		middlewares []MiddlewareFunc
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success":            {fields: fields{middlewares: []MiddlewareFunc{middleware}}},
		"missing middleware": {fields: fields{}, expectedErr: "middlewares are empty"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRawRouteBuilder("/", mockHandler).MethodGet()
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
	mockHandler := func(http.ResponseWriter, *http.Request) {}
	type fields struct {
		authenticator auth.Authenticator
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success":            {fields: fields{authenticator: mockAuth}},
		"missing middleware": {fields: fields{}, expectedErr: "authenticator is nil"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRawRouteBuilder("/", mockHandler).WithAuth(tt.fields.authenticator)

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

func TestRouteBuilder_WithRouteCacheNil(t *testing.T) {
	rb := NewRawRouteBuilder("/", func(writer http.ResponseWriter, request *http.Request) {}).
		WithRouteCache(nil, cache.Age{Max: 1})

	assert.Len(t, rb.errors, 1)
	assert.EqualError(t, rb.errors[0], "route cache is nil")
}

func TestRouteBuilder_Build(t *testing.T) {
	mockAuth := &MockAuthenticator{}
	mockProcessor := func(context.Context, *Request) (*Response, error) { return nil, nil }
	middleware := func(next http.Handler) http.Handler { return next }
	type fields struct {
		path          string
		missingMethod bool
	}
	tests := map[string]struct {
		fields      fields
		expectedErr string
	}{
		"success":           {fields: fields{path: "/"}},
		"missing processor": {fields: fields{path: ""}, expectedErr: "path is empty\n"},
		"missing method":    {fields: fields{path: "/", missingMethod: true}, expectedErr: "method is missing"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRouteBuilder(tt.fields.path, mockProcessor).WithTrace().WithAuth(mockAuth).WithMiddlewares(middleware)
			if !tt.fields.missingMethod {
				rb.MethodGet()
			}
			got, err := rb.Build()

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
		path    string
		handler http.HandlerFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":         {args: args{path: "/", handler: mockHandler}},
		"invalid path":    {args: args{path: "", handler: mockHandler}, expectedErr: "path is empty"},
		"invalid handler": {args: args{path: "/", handler: nil}, expectedErr: "handler is nil"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRawRouteBuilder(tt.args.path, tt.args.handler)

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
	mockProcessor := func(context.Context, *Request) (*Response, error) { return nil, nil }
	type args struct {
		path      string
		processor ProcessorFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":         {args: args{path: "/", processor: mockProcessor}},
		"invalid path":    {args: args{path: "", processor: mockProcessor}, expectedErr: "path is empty"},
		"invalid handler": {args: args{path: "/", processor: nil}, expectedErr: "processor is nil"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rb := NewRouteBuilder(tt.args.path, tt.args.processor)

			if tt.expectedErr != "" {
				assert.Len(t, rb.errors, 1)
				assert.EqualError(t, rb.errors[0], tt.expectedErr)
			} else {
				assert.Len(t, rb.errors, 0)
			}
		})
	}
}

func TestNewGetRouteBuilder(t *testing.T) {
	mockProcessor := func(context.Context, *Request) (*Response, error) { return nil, nil }
	assert.Equal(t, http.MethodGet, NewGetRouteBuilder("/", mockProcessor).method)
}

func TestNewHeadRouteBuilder(t *testing.T) {
	mockProcessor := func(context.Context, *Request) (*Response, error) { return nil, nil }
	assert.Equal(t, http.MethodHead, NewHeadRouteBuilder("/", mockProcessor).method)
}

func TestNewPostRouteBuilder(t *testing.T) {
	mockProcessor := func(context.Context, *Request) (*Response, error) { return nil, nil }
	assert.Equal(t, http.MethodPost, NewPostRouteBuilder("/", mockProcessor).method)
}

func TestNewPutGetRouteBuilder(t *testing.T) {
	mockProcessor := func(context.Context, *Request) (*Response, error) { return nil, nil }
	assert.Equal(t, http.MethodPut, NewPutRouteBuilder("/", mockProcessor).method)
}

func TestNewPatchRouteBuilder(t *testing.T) {
	mockProcessor := func(context.Context, *Request) (*Response, error) { return nil, nil }
	assert.Equal(t, http.MethodPatch, NewPatchRouteBuilder("/", mockProcessor).method)
}

func TestNewDeleteRouteBuilder(t *testing.T) {
	mockProcessor := func(context.Context, *Request) (*Response, error) { return nil, nil }
	assert.Equal(t, http.MethodDelete, NewDeleteRouteBuilder("/", mockProcessor).method)
}

func TestNewConnectRouteBuilder(t *testing.T) {
	mockProcessor := func(context.Context, *Request) (*Response, error) { return nil, nil }
	assert.Equal(t, http.MethodConnect, NewConnectRouteBuilder("/", mockProcessor).method)
}

func TestNewOptionsRouteBuilder(t *testing.T) {
	mockProcessor := func(context.Context, *Request) (*Response, error) { return nil, nil }
	assert.Equal(t, http.MethodOptions, NewOptionsRouteBuilder("/", mockProcessor).method)
}

func TestNewTraceRouteBuilder(t *testing.T) {
	mockProcessor := func(context.Context, *Request) (*Response, error) { return nil, nil }
	assert.Equal(t, http.MethodTrace, NewTraceRouteBuilder("/", mockProcessor).method)
}

func TestRoutesBuilder_Build(t *testing.T) {
	mockHandler := func(http.ResponseWriter, *http.Request) {}
	validRb := NewRawRouteBuilder("/", mockHandler).MethodGet()
	invalidRb := NewRawRouteBuilder("/", mockHandler)
	type args struct {
		rbs []*RouteBuilder
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {
			args: args{rbs: []*RouteBuilder{validRb}},
		},
		"invalid route builder": {
			args:        args{rbs: []*RouteBuilder{invalidRb}},
			expectedErr: "method is missing\n",
		},
		"duplicate routes": {
			args:        args{rbs: []*RouteBuilder{validRb, validRb}},
			expectedErr: "route with key get-/ is duplicate\n",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			builder := NewRoutesBuilder()
			for _, rb := range tt.args.rbs {
				builder.Append(rb)
			}
			got, err := builder.Build()

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.Len(t, got, 1)
			}
		})
	}
}

func TestRoute_Getters(t *testing.T) {
	type testResponse struct {
		Value string
	}

	path := "/foo"
	expectedResponse := testResponse{"foo"}
	r, err := NewRouteBuilder(path, testingHandlerMock(expectedResponse)).WithTrace().MethodPost().Build()
	require.NoError(t, err)

	assert.Equal(t, path, r.Path())
	assert.Equal(t, http.MethodPost, r.Method())
	assert.Len(t, r.Middlewares(), 1)

	// the only way to test do we get the same handler that we provided initially, is to run it explicitly,
	// since all we have in Route itself is a wrapper function
	req, err := http.NewRequest(http.MethodPost, path, nil)
	require.NoError(t, err)
	wr := httptest.NewRecorder()

	r.Handler().ServeHTTP(wr, req)
	br, err := ioutil.ReadAll(wr.Body)
	require.NoError(t, err)

	gotResponse := testResponse{}
	err = json.Unmarshal(br, &gotResponse)
	require.NoError(t, err)

	assert.Equal(t, expectedResponse, gotResponse)
}

func testingHandlerMock(expected interface{}) ProcessorFunc {
	return func(_ context.Context, _ *Request) (*Response, error) {
		return NewResponse(expected), nil
	}
}
