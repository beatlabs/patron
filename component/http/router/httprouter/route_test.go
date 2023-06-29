package httprouter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileServerRoute(t *testing.T) {
	type args struct {
		path         string
		assetsDir    string
		fallbackPath string
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{
			path:         "/frontend/*path",
			assetsDir:    "testdata/",
			fallbackPath: "testdata/index.html",
		}},
		"missing path": {args: args{
			path:         "",
			assetsDir:    "123",
			fallbackPath: "123",
		}, expectedErr: "path is empty"},
		"missing assets": {args: args{
			path:         "123",
			assetsDir:    "",
			fallbackPath: "123",
		}, expectedErr: "assets path is empty"},
		"missing fallback path": {args: args{
			path:         "123",
			assetsDir:    "123",
			fallbackPath: "",
		}, expectedErr: "fallback path is empty"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			got, err := NewFileServerRoute(tt.args.path, tt.args.assetsDir, tt.args.fallbackPath)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, "/frontend/*path", got.Path())
				assert.Equal(t, http.MethodGet, got.Method())
				assert.NotNil(t, got.Handler())
				assert.Len(t, got.Middlewares(), 0)
			}
		})
	}
}

func TestFileServerRouteHandler(t *testing.T) {
	handler, err := NewFileServerRoute("/frontend/*path", "testdata/", "testdata/index.html")
	require.NoError(t, err)

	type args struct {
		path string
	}
	tests := map[string]struct {
		args         args
		expectedCode int
		expectedErr  string
	}{
		"fallback": {args: args{path: "frontend"}, expectedCode: 200},
		"index":    {args: args{path: "frontend/index"}, expectedCode: 200},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, tt.args.path, nil)
			require.NoError(t, err)
			rc := httptest.NewRecorder()
			handler.Handler().ServeHTTP(rc, req)
			assert.Equal(t, tt.expectedCode, rc.Code)
		})
	}
}
