package patron

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/beatlabs/patron/log/std"

	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/log"
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
		"provided SIGHUP handler was nil\n"

	tests := map[string]struct {
		fields        map[string]interface{}
		cps           []Component
		routesBuilder *patronhttp.RoutesBuilder
		middlewares   []patronhttp.MiddlewareFunc
		acf           patronhttp.AliveCheckFunc
		rcf           patronhttp.ReadyCheckFunc
		sighupHandler func()
		wantErr       string
	}{
		"success": {
			fields:        map[string]interface{}{"env": "dev"},
			cps:           []Component{&testComponent{}, &testComponent{}},
			routesBuilder: routesBuilder,
			middlewares:   []patronhttp.MiddlewareFunc{middleware},
			acf:           patronhttp.DefaultAliveCheck,
			rcf:           patronhttp.DefaultReadyCheck,
			sighupHandler: func() { log.Info("SIGHUP received: nothing setup") },
			wantErr:       "",
		},
		"nil inputs steps": {
			cps:           nil,
			routesBuilder: nil,
			middlewares:   nil,
			acf:           nil,
			rcf:           nil,
			sighupHandler: nil,
			wantErr:       httpBuilderAllErrors,
		},
		"error in all builder steps": {
			cps:           []Component{},
			routesBuilder: nil,
			middlewares:   []patronhttp.MiddlewareFunc{},
			acf:           nil,
			rcf:           nil,
			sighupHandler: nil,
			wantErr:       httpBuilderAllErrors,
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
			err := os.Setenv("PATRON_HTTP_DEFAULT_PORT", getRandomPort())
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
		name string
		cp   Component
		host string
		port string
	}{
		{name: "success w/ empty tracing vars", cp: &testComponent{}},
		{name: "success w/ empty tracing host", cp: &testComponent{}, port: "6831"},
		{name: "success w/ empty tracing port", cp: &testComponent{}, host: "127.0.0.1"},
		{name: "success", cp: &testComponent{}, host: "127.0.0.1", port: "6831"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.host != "" {
				err := os.Setenv("PATRON_JAEGER_AGENT_HOST", tt.host)
				assert.NoError(t, err)
			}
			if tt.port != "" {
				err := os.Setenv("PATRON_JAEGER_AGENT_PORT", tt.port)
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
		samplerParam string
		port         string
		expectedErr  string
	}{
		"success with wrong w/ port":             {port: "foo"},
		"success with wrong w/ overflowing port": {port: "153000"},
		"failure w/ sampler param":               {samplerParam: "foo", expectedErr: "env var for jaeger sampler param is not valid: strconv.ParseFloat: parsing \"foo\": invalid syntax"},
		"failure w/ overflowing sampler param":   {samplerParam: "8", expectedErr: "cannot initialize jaeger tracer: invalid Param for probabilistic sampler; expecting value between 0 and 1, received 8"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.samplerParam != "" {
				err := os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", tt.samplerParam)
				assert.NoError(t, err)
			}
			if tt.port != "" {
				err := os.Setenv("PATRON_HTTP_DEFAULT_PORT", tt.port)
				assert.NoError(t, err)
			}
			svc, err := New("test", "", TextLogger())
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, svc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, svc)
			}
		})
		err := os.Unsetenv("PATRON_JAEGER_SAMPLER_PARAM")
		require.NoError(t, err)

		err = os.Unsetenv("PATRON_HTTP_DEFAULT_PORT")
		require.NoError(t, err)
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

func getRandomPort() string {
	rnd := 50000 + rand.Int63n(10000)
	return strconv.FormatInt(rnd, 10)
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
