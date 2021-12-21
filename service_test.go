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

	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/log/std"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	routesBuilder := patronhttp.NewRoutesBuilder().
		Append(patronhttp.NewRawRouteBuilder("/", func(w http.ResponseWriter, r *http.Request) {}).MethodGet())

	middleware := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	}

	httpBuilderAllErrors := "routes builder is nil\n" +
		"provided middlewares slice was empty\n" +
		"alive check func provided was nil\n" +
		"ready check func provided was nil\n" +
		"provided components slice was empty\n" +
		"provided SIGHUP handler was nil\n" +
		"provided uncompressed paths slice was empty\n"

	tests := map[string]struct {
		fields            map[string]interface{}
		cps               []Component
		routesBuilder     *patronhttp.RoutesBuilder
		middlewares       []patronhttp.MiddlewareFunc
		acf               patronhttp.AliveCheckFunc
		rcf               patronhttp.ReadyCheckFunc
		sighupHandler     func()
		uncompressedPaths []string
		wantErr           string
	}{
		"success": {
			fields:            map[string]interface{}{"env": "dev"},
			cps:               []Component{&testComponent{}, &testComponent{}},
			routesBuilder:     routesBuilder,
			middlewares:       []patronhttp.MiddlewareFunc{middleware},
			acf:               patronhttp.DefaultAliveCheck,
			rcf:               patronhttp.DefaultReadyCheck,
			sighupHandler:     func() { log.Info("SIGHUP received: nothing setup") },
			uncompressedPaths: []string{"/foo", "/bar"},
			wantErr:           "",
		},
		"nil inputs steps": {
			cps:               nil,
			routesBuilder:     nil,
			middlewares:       nil,
			acf:               nil,
			rcf:               nil,
			sighupHandler:     nil,
			uncompressedPaths: nil,
			wantErr:           httpBuilderAllErrors,
		},
		"error in all builder steps": {
			cps:               []Component{},
			routesBuilder:     nil,
			middlewares:       []patronhttp.MiddlewareFunc{},
			acf:               nil,
			rcf:               nil,
			sighupHandler:     nil,
			uncompressedPaths: []string{},
			wantErr:           httpBuilderAllErrors,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			svc, err := New("name", "1.0", LogFields(tt.fields), TextLogger())
			require.NoError(t, err)
			gotService, gotErr := svc.
				WithRoutesBuilder(tt.routesBuilder).
				WithMiddlewares(tt.middlewares...).
				WithAliveCheck(tt.acf).
				WithReadyCheck(tt.rcf).
				WithComponents(tt.cps...).
				WithSIGHUP(tt.sighupHandler).
				WithUncompressedPaths(tt.uncompressedPaths...).
				build()

			if tt.wantErr != "" {
				assert.EqualError(t, gotErr, tt.wantErr)
				assert.Nil(t, gotService)
			} else {
				assert.Nil(t, gotErr)
				assert.NotNil(t, gotService)
				assert.IsType(t, &service{}, gotService)

				assert.NotEmpty(t, gotService.cps)
				assert.NotNil(t, gotService.routesBuilder)
				assert.Len(t, gotService.middlewares, len(tt.middlewares))
				assert.NotNil(t, gotService.rcf)
				assert.NotNil(t, gotService.acf)
				assert.NotNil(t, gotService.termSig)
				assert.NotNil(t, gotService.sighupHandler)

				for _, comp := range tt.cps {
					assert.Contains(t, gotService.cps, comp)
				}

				for _, middleware := range tt.middlewares {
					assert.NotNil(t, middleware)
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
		t.Run(name, func(t *testing.T) {
			defer os.Clearenv()

			err := os.Setenv("PATRON_HTTP_DEFAULT_PORT", getRandomPort(t))
			assert.NoError(t, err)
			svc, err := New("test", "", TextLogger())
			require.NoError(t, err)
			err = svc.WithComponents(tt.cp, tt.cp, tt.cp).Run(context.Background())
			if tt.wantErr {
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
		t.Run(tt.name, func(t *testing.T) {
			defer os.Clearenv()

			if tt.host != "" {
				err := os.Setenv("PATRON_JAEGER_AGENT_HOST", tt.host)
				assert.NoError(t, err)
			}
			if tt.port != "" {
				err := os.Setenv("PATRON_JAEGER_AGENT_PORT", tt.port)
				assert.NoError(t, err)
			}
			if tt.buckets != "" {
				err := os.Setenv("PATRON_JAEGER_DEFAULT_BUCKETS", tt.buckets)
				assert.NoError(t, err)
			}
			svc, err := New("test", "", TextLogger())
			require.NoError(t, err)
			s, err := svc.WithComponents(tt.cp, tt.cp, tt.cp).build()
			assert.NoError(t, err)
			err = s.run(context.Background())
			assert.NoError(t, err)
		})
	}
}

func TestBuilder_WithComponentsTwice(t *testing.T) {
	svc, err := New("test", "", TextLogger())
	require.NoError(t, err)
	bld := svc.WithComponents(&testComponent{}).WithComponents(&testComponent{})
	assert.Len(t, bld.cps, 2)
}

func TestBuild_FailingConditions(t *testing.T) {
	tests := map[string]struct {
		jaegerSamplerParam string
		port               string
		jaegerBuckets      string
		expectedBuildErr   string
		expectedRunErr     string
	}{
		"failure with wrong w/ port":             {port: "foo", expectedRunErr: "env var for HTTP default port is not valid: strconv.ParseInt: parsing \"foo\": invalid syntax"},
		"success with wrong w/ overflowing port": {port: "153000", expectedRunErr: "failed to create default HTTP component: invalid HTTP Port provided\n"},
		"failure w/ sampler param":               {jaegerSamplerParam: "foo", expectedRunErr: "env var for jaeger sampler param is not valid: strconv.ParseFloat: parsing \"foo\": invalid syntax"},
		"failure w/ overflowing sampler param":   {jaegerSamplerParam: "8", expectedRunErr: "cannot initialize jaeger tracer: invalid Param for probabilistic sampler; expecting value between 0 and 1, received 8"},
		"failure w/ custom default buckets":      {jaegerSamplerParam: "1", jaegerBuckets: "foo", expectedRunErr: "env var for jaeger default buckets contains invalid value: strconv.ParseFloat: parsing \"foo\": invalid syntax"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			defer os.Clearenv()

			if tt.port != "" {
				err := os.Setenv("PATRON_HTTP_DEFAULT_PORT", tt.port)
				require.NoError(t, err)
			}
			if tt.jaegerSamplerParam != "" {
				err := os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", tt.jaegerSamplerParam)
				require.NoError(t, err)
			}
			if tt.jaegerBuckets != "" {
				err := os.Setenv("PATRON_JAEGER_DEFAULT_BUCKETS", tt.jaegerBuckets)
				require.NoError(t, err)
			}

			svc, err := New("test", "", TextLogger())
			if tt.expectedBuildErr != "" {
				require.EqualError(t, err, tt.expectedBuildErr)
				require.Nil(t, svc)
			} else {
				require.NoError(t, err)
				require.NotNil(t, svc)
			}

			// start running with a canceled context, on purpose
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			err = svc.Run(ctx)

			if tt.expectedRunErr != "" {
				require.EqualError(t, err, tt.expectedRunErr)
			} else {
				require.Equal(t, err, context.Canceled)
			}
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
		t.Run(tt.name, func(t *testing.T) {
			defer os.Clearenv()

			if tt.rt != "" {
				err := os.Setenv("PATRON_HTTP_READ_TIMEOUT", tt.rt)
				assert.NoError(t, err)
			}
			if tt.wt != "" {
				err := os.Setenv("PATRON_HTTP_WRITE_TIMEOUT", tt.wt)
				assert.NoError(t, err)
			}
			svc, err := New("test", "", TextLogger())
			require.NoError(t, err)

			_, err = svc.WithComponents(tt.cp, tt.cp, tt.cp).build()
			if tt.wantErr {
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
		t.Run(tt.name, func(t *testing.T) {
			defer os.Clearenv()

			if tt.level != "" {
				err := os.Setenv("PATRON_COMPRESSION_DEFLATE_LEVEL", tt.level)
				assert.NoError(t, err)
			}
			svc, err := New("test", "", TextLogger())
			require.NoError(t, err)

			_, err = svc.WithComponents(tt.component, tt.component, tt.component).build()
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

func TestLogFields(t *testing.T) {
	defaultFields := defaultLogFields("test", "1.0")
	fields := map[string]interface{}{"key": "value"}
	fields1 := defaultLogFields("name1", "version1")
	type args struct {
		fields map[string]interface{}
	}
	tests := map[string]struct {
		args args
		want Config
	}{
		"success":      {args: args{fields: fields}, want: Config{fields: mergeFields(defaultFields, fields)}},
		"no overwrite": {args: args{fields: fields1}, want: Config{fields: defaultFields}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := Config{fields: defaultFields}
			LogFields(tt.args.fields)(&cfg)
			assert.Equal(t, tt.want, cfg)
		})
	}
}

func mergeFields(ff1, ff2 map[string]interface{}) map[string]interface{} {
	ff := map[string]interface{}{}
	for k, v := range ff1 {
		ff[k] = v
	}
	for k, v := range ff2 {
		ff[k] = v
	}
	return ff
}

func TestLogger(t *testing.T) {
	logger := std.New(os.Stderr, getLogLevel(), nil)
	cfg := Config{}
	Logger(logger)(&cfg)
	assert.Equal(t, logger, cfg.logger)
}
