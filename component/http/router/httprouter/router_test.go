package httprouter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	route, err := patronhttp.NewRoute(http.MethodGet, "/api/", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
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
		tt := tt
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
	route, err := patronhttp.NewRoute(http.MethodGet, "/api/", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
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

	t.Run("check metrics endpoint", func(t *testing.T) {
		rsp, err := http.Get(srv.URL + "/metrics")
		require.NoError(t, err)
		assertResponse(t, rsp)
	})

	t.Run("check alive endpoint", func(t *testing.T) {
		rsp, err := http.Get(srv.URL + "/alive")
		require.NoError(t, err)
		assertResponse(t, rsp)
	})

	t.Run("check alive endpoint", func(t *testing.T) {
		rsp, err := http.Get(srv.URL + "/ready")
		require.NoError(t, err)
		assertResponse(t, rsp)
	})

	t.Run("check pprof endpoint", func(t *testing.T) {
		rsp, err := http.Get(srv.URL + "/debug/pprof")
		require.NoError(t, err)
		assertResponse(t, rsp)
	})

	t.Run("check provided endpoint", func(t *testing.T) {
		rsp, err := http.Get(srv.URL + "/api")
		require.NoError(t, err)
		assertResponse(t, rsp)
	})
}
