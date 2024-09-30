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
		assertResponse(t, rsp)
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
