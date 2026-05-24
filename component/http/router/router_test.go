package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	route, err := patronhttp.NewRoute("GET /api/", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	require.NoError(t, err)
	type args struct {
		oo []OptionFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":            {args: args{oo: []OptionFunc{WithRoutes(route)}}},
		"option func failed": {args: args{oo: []OptionFunc{WithAliveCheck(nil)}}, expectedErr: "alive check function is nil"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, err := New(tt.args.oo...)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NotNil(t, got)
			}
		})
	}
}

func TestVerifyRouter(t *testing.T) {
	route, err := patronhttp.NewRoute("GET /api/", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	require.NoError(t, err)

	appName := "appName"
	appVersion := "1.1"
	appVersionHeader := "X-App-Version"
	appNameHeader := "X-App-Name"

	appNameVersionOptionFunc, err := WithAppNameHeaders(appName, appVersion)
	require.NoError(t, err)

	router, err := New(WithRoutes(route), appNameVersionOptionFunc)
	require.NoError(t, err)

	srv := httptest.NewServer(router)
	defer func() {
		srv.Close()
	}()

	assertResponse := func(t *testing.T, rsp *http.Response) {
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
		assert.Equal(t, appName, rsp.Header.Get(appNameHeader))
		assert.Equal(t, appVersion, rsp.Header.Get(appVersionHeader))
	}

	t.Run("check alive endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/alive", nil)
		require.NoError(t, err)
		rsp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assertResponse(t, rsp)
		require.NoError(t, rsp.Body.Close())
	})

	t.Run("check alive endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/ready", nil)
		require.NoError(t, err)
		rsp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assertResponse(t, rsp)
		require.NoError(t, rsp.Body.Close())
	})

	t.Run("check pprof endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/debug/pprof", nil)
		require.NoError(t, err)
		rsp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rsp.StatusCode)
		require.NoError(t, rsp.Body.Close())
	})

	t.Run("check provided endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/api", nil)
		require.NoError(t, err)
		rsp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assertResponse(t, rsp)
		require.NoError(t, rsp.Body.Close())
	})
}

func TestProfiling(t *testing.T) {
	router, err := New(WithProfiling())
	require.NoError(t, err)

	srv := httptest.NewServer(router)
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/debug/pprof", nil)
	require.NoError(t, err)
	rsp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rsp.StatusCode)
	require.NoError(t, rsp.Body.Close())
}

func TestProfilingMiddlewares(t *testing.T) {
	deny := func(_ http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
	}

	router, err := New(WithProfilingMiddlewares(deny))
	require.NoError(t, err)

	srv := httptest.NewServer(router)
	defer srv.Close()

	t.Run("protects pprof endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/debug/pprof", nil)
		require.NoError(t, err)
		rsp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rsp.StatusCode)
		require.NoError(t, rsp.Body.Close())
	})

	t.Run("does not protect alive endpoint", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/alive", nil)
		require.NoError(t, err)
		rsp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rsp.StatusCode)
		require.NoError(t, rsp.Body.Close())
	})
}

func TestProfilingMiddlewaresOption(t *testing.T) {
	cfg := &Config{}
	err := WithProfilingMiddlewares(func(next http.Handler) http.Handler { return next })(cfg)
	require.NoError(t, err)
	assert.True(t, cfg.enableProfiling)
	assert.Len(t, cfg.profilingMiddlewares, 1)

	err = WithProfilingMiddlewares()(cfg)
	assert.EqualError(t, err, "middlewares are empty")
}
