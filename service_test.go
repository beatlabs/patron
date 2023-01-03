package patron

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/beatlabs/patron/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	mw := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	}

	httpBuilderAllErrors := "provided components slice was empty\n" +
		"provided WithSIGHUP handler was nil\n" +
		"provided router is nil\n"

	tests := map[string]struct {
		fields            map[string]interface{}
		cps               []Component
		sighupHandler     func()
		uncompressedPaths []string
		handler           http.Handler
		wantErr           string
	}{
		"success": {
			fields:            map[string]interface{}{"env": "dev"},
			cps:               []Component{&testComponent{}, &testComponent{}},
			sighupHandler:     func() { log.Info("WithSIGHUP received: nothing setup") },
			uncompressedPaths: []string{"/foo", "/bar"},
			handler:           mw(nil),
			wantErr:           "",
		},
		"nil inputs steps": {
			cps:               nil,
			sighupHandler:     nil,
			uncompressedPaths: nil,
			handler:           nil,
			wantErr:           httpBuilderAllErrors,
		},
		"error in all builder steps": {
			cps:               []Component{},
			sighupHandler:     nil,
			uncompressedPaths: []string{},
			handler:           nil,
			wantErr:           httpBuilderAllErrors,
		},
	}

	for name, tt := range tests {
		temp := tt
		t.Run(name, func(t *testing.T) {
			gotService, gotErr := New("name", "1.0", WithLogFields(temp.fields), WithTextLogger(),
				WithComponents(temp.cps...), WithSIGHUP(temp.sighupHandler), WithRouter(temp.handler))

			if temp.wantErr != "" {
				assert.EqualError(t, gotErr, temp.wantErr)
				assert.Nil(t, gotService)
			} else {
				assert.Nil(t, gotErr)
				assert.NotNil(t, gotService)
				assert.IsType(t, &Service{}, gotService)

				assert.NotEmpty(t, gotService.cps)
				assert.NotNil(t, gotService.termSig)
				assert.NotNil(t, gotService.sighupHandler)

				for _, comp := range temp.cps {
					assert.Contains(t, gotService.cps, comp)
				}
			}
		})
	}
}

func TestServer_Run_Shutdown(t *testing.T) {
	tests := map[string]struct {
		cp      Component
		wantErr bool
	}{
		"success":       {cp: &testComponent{}, wantErr: false},
		"failed to run": {cp: &testComponent{errorRunning: true}, wantErr: true},
	}
	for name, tt := range tests {
		temp := tt
		t.Run(name, func(t *testing.T) {
			defer func() {
				os.Clearenv()
			}()
			t.Setenv("PATRON_HTTP_DEFAULT_PORT", getRandomPort(t))
			svc, err := New("test", "", WithTextLogger(), WithComponents(temp.cp, temp.cp, temp.cp))
			assert.NoError(t, err)
			err = svc.Run(context.Background())
			if temp.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_SetupTracing(t *testing.T) {
	tests := []struct {
		name    string
		cp      Component
		host    string
		port    string
		buckets string
	}{
		{name: "success w/ empty tracing vars", cp: &testComponent{}},
		{name: "success w/ empty tracing host", cp: &testComponent{}, port: "6831"},
		{name: "success w/ empty tracing port", cp: &testComponent{}, host: "127.0.0.1"},
		{name: "success", cp: &testComponent{}, host: "127.0.0.1", port: "6831"},
		{name: "success w/ custom default buckets", cp: &testComponent{}, host: "127.0.0.1", port: "6831", buckets: ".1, .3"},
	}
	for _, tt := range tests {
		temp := tt
		t.Run(temp.name, func(t *testing.T) {
			defer os.Clearenv()

			if temp.host != "" {
				err := os.Setenv("PATRON_JAEGER_AGENT_HOST", temp.host)
				assert.NoError(t, err)
			}
			if temp.port != "" {
				err := os.Setenv("PATRON_JAEGER_AGENT_PORT", temp.port)
				assert.NoError(t, err)
			}
			if temp.buckets != "" {
				err := os.Setenv("PATRON_JAEGER_DEFAULT_BUCKETS", temp.buckets)
				assert.NoError(t, err)
			}

			svc, err := New("test", "", WithTextLogger(), WithComponents(tt.cp, tt.cp, tt.cp))
			assert.NoError(t, err)

			err = svc.Run(context.Background())
			assert.NoError(t, err)
		})
	}
}

func TestNewServer_WithComponentsTwice(t *testing.T) {
	svc, err := New("test", "", WithTextLogger(), WithComponents(&testComponent{}, &testComponent{}))
	require.NoError(t, err)
	assert.Len(t, svc.cps, 3)
}

func TestNewServer_FailingConditions(t *testing.T) {
	tests := map[string]struct {
		jaegerSamplerParam       string
		port                     string
		jaegerBuckets            string
		expectedConstructorError string
	}{
		"failure with wrong w/ port":             {port: "foo", expectedConstructorError: "env var for HTTP default port is not valid: strconv.ParseInt: parsing \"foo\": invalid syntax"},
		"success with wrong w/ overflowing port": {port: "153000", expectedConstructorError: "invalid HTTP Port provided"},
		"failure w/ sampler param":               {jaegerSamplerParam: "foo", expectedConstructorError: "env var for jaeger sampler param is not valid: strconv.ParseFloat: parsing \"foo\": invalid syntax"},
		"failure w/ overflowing sampler param":   {jaegerSamplerParam: "8", expectedConstructorError: "cannot initialize jaeger tracer: invalid Param for probabilistic sampler; expecting value between 0 and 1, received 8"},
		"failure w/ custom default buckets":      {jaegerSamplerParam: "1", jaegerBuckets: "foo", expectedConstructorError: "env var for jaeger default buckets contains invalid value: strconv.ParseFloat: parsing \"foo\": invalid syntax"},
	}

	for name, tt := range tests {
		temp := tt
		t.Run(name, func(t *testing.T) {
			defer os.Clearenv()

			if temp.port != "" {
				err := os.Setenv("PATRON_HTTP_DEFAULT_PORT", temp.port)
				require.NoError(t, err)
			}
			if temp.jaegerSamplerParam != "" {
				err := os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", temp.jaegerSamplerParam)
				require.NoError(t, err)
			}
			if temp.jaegerBuckets != "" {
				err := os.Setenv("PATRON_JAEGER_DEFAULT_BUCKETS", temp.jaegerBuckets)
				require.NoError(t, err)
			}

			svc, err := New("test", "", WithTextLogger())

			if temp.expectedConstructorError != "" {
				require.EqualError(t, err, temp.expectedConstructorError)
				require.Nil(t, svc)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, svc)

			// start running with a canceled context, on purpose
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err = svc.Run(ctx)
			require.NoError(t, err)

			require.Equal(t, err, context.Canceled)
		})
	}
}

func TestServer_SetupReadWriteTimeouts(t *testing.T) {
	tests := []struct {
		name    string
		cp      Component
		ctx     context.Context
		rt      string
		wt      string
		wantErr bool
	}{
		{name: "success wo/ setup read and write timeouts", cp: &testComponent{}, ctx: context.Background(), wantErr: false},
		{name: "success w/ setup read and write timeouts", cp: &testComponent{}, ctx: context.Background(), rt: "60s", wt: "20s", wantErr: false},
		{name: "failed w/ invalid write timeout", cp: &testComponent{}, ctx: context.Background(), wt: "invalid", wantErr: true},
		{name: "failed w/ invalid read timeout", cp: &testComponent{}, ctx: context.Background(), rt: "invalid", wantErr: true},
		{name: "failed w/ negative write timeout", cp: &testComponent{}, ctx: context.Background(), wt: "-100s", wantErr: true},
		{name: "failed w/ zero read timeout", cp: &testComponent{}, ctx: context.Background(), rt: "0s", wantErr: true},
	}
	for _, tt := range tests {
		temp := tt
		t.Run(temp.name, func(t *testing.T) {
			defer os.Clearenv()

			if temp.rt != "" {
				err := os.Setenv("PATRON_HTTP_READ_TIMEOUT", temp.rt)
				assert.NoError(t, err)
			}
			if temp.wt != "" {
				err := os.Setenv("PATRON_HTTP_WRITE_TIMEOUT", temp.wt)
				assert.NoError(t, err)
			}
			_, err := New("test", "", WithTextLogger(), WithComponents(temp.cp, temp.cp, temp.cp))

			if temp.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_SetupDeflateLevel(t *testing.T) {
	tests := []struct {
		name      string
		component Component
		ctx       context.Context
		level     string
		wantErr   bool
	}{
		{name: "success without setup compression deflate level", component: &testComponent{}, ctx: context.Background(), wantErr: false},
		{name: "success with setup compression deflate level = -2", component: &testComponent{}, ctx: context.Background(), level: "-2", wantErr: false},
		{name: "success with setup compression deflate level = 2", component: &testComponent{}, ctx: context.Background(), level: "2", wantErr: false},
		{name: "success with setup compression deflate level = 6", component: &testComponent{}, ctx: context.Background(), level: "6", wantErr: false},
		{name: "success with setup compression deflate level = 9", component: &testComponent{}, ctx: context.Background(), level: "9", wantErr: false},
		{name: "failed with too small compression deflate level", component: &testComponent{}, ctx: context.Background(), level: "-3", wantErr: true},
		{name: "failed with too big compression deflate level", component: &testComponent{}, ctx: context.Background(), level: "10", wantErr: true},
		{name: "failed with invalid compression deflate level", component: &testComponent{}, ctx: context.Background(), level: "blah", wantErr: true},
	}
	for _, tt := range tests {
		temp := tt
		t.Run(tt.name, func(t *testing.T) {
			defer os.Clearenv()

			if temp.level != "" {
				err := os.Setenv("PATRON_COMPRESSION_DEFLATE_LEVEL", temp.level)
				assert.NoError(t, err)
			}

			_, err := New("test", "", WithTextLogger(), WithComponents(temp.component, temp.component, temp.component))

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func getRandomPort(t *testing.T) string {
	bg, err := rand.Int(rand.Reader, big.NewInt(10000))
	require.NoError(t, err)
	return strconv.FormatInt(bg.Int64(), 10)
}

type testComponent struct {
	errorRunning bool
}

func (ts testComponent) Run(_ context.Context) error {
	if ts.errorRunning {
		return errors.New("failed to run component")
	}
	return nil
}
