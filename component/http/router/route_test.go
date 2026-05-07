package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRoutePath    = "GET /frontend/*path"
	testAssetsDir    = "testdata/"
	testFallbackPath = "testdata/index.html"
	testURLPath      = "/frontend/x"
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
			path:         testRoutePath,
			assetsDir:    testAssetsDir,
			fallbackPath: testFallbackPath,
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
			path:         testRoutePath,
			assetsDir:    "nonexistent_dir",
			fallbackPath: testFallbackPath,
		}, expectedErr: "assets directory [GET /frontend/*path] doesn't exist"},
		"nonexistent fallback file": {args: args{
			path:         testRoutePath,
			assetsDir:    testAssetsDir,
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
				assert.Equal(t, testRoutePath, got.Path())
				assert.NotNil(t, got.Handler())
				assert.Empty(t, got.Middlewares())
			}
		})
	}
}

func TestFileServerRouteHandler(t *testing.T) {
	handler, err := NewFileServerRoute(testRoutePath, testAssetsDir, testFallbackPath)
	require.NoError(t, err)

	tests := map[string]struct {
		urlPath      string // request URL (must be clean so http.ServeFile accepts it)
		pathValue    string // value PathValue("path") returns, as set by the router
		expectedCode int
	}{
		"fallback": {urlPath: "/frontend/", pathValue: "", expectedCode: 200},
		// urlPath must not end in /index.html — http.ServeFile redirects that to ./
		"index":             {urlPath: "/frontend/app", pathValue: "index.html", expectedCode: 200},
		"file not found":    {urlPath: testURLPath, pathValue: "nonexistent.html", expectedCode: 200},
		"traversal attempt": {urlPath: testURLPath, pathValue: "../../etc/passwd", expectedCode: 200},
		"traversal nested":  {urlPath: testURLPath, pathValue: "sub/../../etc/passwd", expectedCode: 200},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, tt.urlPath, nil)
			req.SetPathValue("path", tt.pathValue)
			rc := httptest.NewRecorder()
			handler.Handler().ServeHTTP(rc, req)
			assert.Equal(t, tt.expectedCode, rc.Code)
		})
	}
}
