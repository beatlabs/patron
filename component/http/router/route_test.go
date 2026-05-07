package router

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
			path:         "GET /frontend/*path",
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
		"nonexistent assets dir": {args: args{
			path:         "GET /frontend/*path",
			assetsDir:    "nonexistent_dir",
			fallbackPath: "testdata/index.html",
		}, expectedErr: "assets directory [GET /frontend/*path] doesn't exist"},
		"nonexistent fallback file": {args: args{
			path:         "GET /frontend/*path",
			assetsDir:    "testdata/",
			fallbackPath: "nonexistent.html",
		}, expectedErr: "fallback file [nonexistent.html] doesn't exist"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := NewFileServerRoute(tt.args.path, tt.args.assetsDir, tt.args.fallbackPath)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, "GET /frontend/*path", got.Path())
				assert.NotNil(t, got.Handler())
				assert.Empty(t, got.Middlewares())
			}
		})
	}
}

func TestFileServerRouteHandler(t *testing.T) {
	handler, err := NewFileServerRoute("GET /frontend/*path", "testdata/", "testdata/index.html")
	require.NoError(t, err)

	tests := map[string]struct {
		urlPath      string // request URL (must be clean so http.ServeFile accepts it)
		pathValue    string // value PathValue("path") returns, as set by the router
		expectedCode int
	}{
		"fallback": {urlPath: "/frontend/", pathValue: "", expectedCode: 200},
		// urlPath must not end in /index.html — http.ServeFile redirects that to ./
		"index":             {urlPath: "/frontend/app", pathValue: "index.html", expectedCode: 200},
		"file not found":    {urlPath: "/frontend/x", pathValue: "nonexistent.html", expectedCode: 200},
		"traversal attempt": {urlPath: "/frontend/x", pathValue: "../../etc/passwd", expectedCode: 200},
		"traversal nested":  {urlPath: "/frontend/x", pathValue: "sub/../../etc/passwd", expectedCode: 200},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.urlPath, nil)
			req.SetPathValue("path", tt.pathValue)
			rc := httptest.NewRecorder()
			handler.Handler().ServeHTTP(rc, req)
			assert.Equal(t, tt.expectedCode, rc.Code)
		})
	}
}
