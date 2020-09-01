package patron

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"

	phttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/log"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	routesBuilder := phttp.NewRoutesBuilder().
		Append(phttp.NewRawRouteBuilder("/", func(w http.ResponseWriter, r *http.Request) {}).MethodGet())

	middleware := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	}

	httpBuilderAllErrors := errors.New("name is required\n" +
		"routes builder is nil\n" +
		"provided middlewares slice was empty\n" +
		"alive check func provided was nil\n" +
		"ready check func provided was nil\n" +
		"provided components slice was empty\n" +
		"provided SIGHUP handler was nil\n")

	tests := map[string]struct {
		name          string
		version       string
		fields        map[string]interface{}
		cps           []Component
		routesBuilder *phttp.RoutesBuilder
		middlewares   []phttp.MiddlewareFunc
		acf           phttp.AliveCheckFunc
		rcf           phttp.ReadyCheckFunc
		sighupHandler func()
		wantErr       error
	}{
		"success": {
			name:          "test",
			version:       "dev",
			fields:        map[string]interface{}{"env": "dev"},
			cps:           []Component{&testComponent{}, &testComponent{}},
			routesBuilder: routesBuilder,
			middlewares:   []phttp.MiddlewareFunc{middleware},
			acf:           phttp.DefaultAliveCheck,
			rcf:           phttp.DefaultReadyCheck,
			sighupHandler: func() { log.Info("SIGHUP received: nothing setup") },
			wantErr:       nil,
		},
		"nil inputs steps": {
			name:          "",
			version:       "",
			cps:           nil,
			routesBuilder: nil,
			middlewares:   nil,
			acf:           nil,
			rcf:           nil,
			sighupHandler: nil,
			wantErr:       httpBuilderAllErrors,
		},
		"error in all builder steps": {
			name:          "",
			version:       "",
			cps:           []Component{},
			routesBuilder: nil,
			middlewares:   []phttp.MiddlewareFunc{},
			acf:           nil,
			rcf:           nil,
			sighupHandler: nil,
			wantErr:       httpBuilderAllErrors,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotService, gotErr := New(tt.name, tt.version).
				WithRoutesBuilder(tt.routesBuilder).
				WithMiddlewares(tt.middlewares...).
				WithLogFields(tt.fields).
				WithAliveCheck(tt.acf).
				WithReadyCheck(tt.rcf).
				WithComponents(tt.cps...).
				WithSIGHUP(tt.sighupHandler).
				build()

			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr.Error(), gotErr.Error())
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
	tests := []struct {
		name    string
		cp      Component
		ctx     context.Context
		wantErr bool
	}{
		{name: "success", cp: &testComponent{}, ctx: context.Background(), wantErr: false},
		{name: "failed to run", cp: &testComponent{errorRunning: true}, ctx: context.Background(), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := os.Setenv("PATRON_HTTP_DEFAULT_PORT", getRandomPort())
			assert.NoError(t, err)
			err = New("test", "").WithComponents(tt.cp, tt.cp, tt.cp).Run(tt.ctx)
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
		ctx  context.Context
		host string
		port string
	}{
		{name: "success w/ empty tracing vars", cp: &testComponent{}, ctx: context.Background()},
		{name: "success w/ empty tracing host", cp: &testComponent{}, ctx: context.Background(), port: "6831"},
		{name: "success w/ empty tracing port", cp: &testComponent{}, ctx: context.Background(), host: "127.0.0.1"},
		{name: "success", cp: &testComponent{}, ctx: context.Background(), host: "127.0.0.1", port: "6831"},
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
			s, err := New("test", "").WithComponents(tt.cp, tt.cp, tt.cp).build()
			assert.NoError(t, err)
			err = s.run(tt.ctx)
			assert.NoError(t, err)
		})
	}
}

func TestBuilder_WithComponentsTwice(t *testing.T) {
	bld := New("test", "").WithComponents(&testComponent{}).WithComponents(&testComponent{})
	assert.Len(t, bld.cps, 2)
}

func TestBuild_FailingConditions(t *testing.T) {
	tests := []struct {
		name         string
		cp           Component
		ctx          context.Context
		samplerParam string
		port         string
	}{
		{name: "failure w/ port", cp: &testComponent{}, ctx: context.Background(), port: "foo"},
		{name: "failure w/ overflowing port", cp: &testComponent{}, ctx: context.Background(), port: "153000"},
		{name: "failure w/ sampler param", cp: &testComponent{}, ctx: context.Background(), samplerParam: "foo"},
		{name: "failure w/ overflowing sampler param", cp: &testComponent{}, ctx: context.Background(), samplerParam: "8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.samplerParam != "" {
				err := os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", tt.samplerParam)
				assert.NoError(t, err)
			}
			if tt.port != "" {
				err := os.Setenv("PATRON_HTTP_DEFAULT_PORT", tt.port)
				assert.NoError(t, err)
			}
			err := New("test", "").WithComponents(tt.cp, tt.cp, tt.cp).Run(tt.ctx)
			assert.Error(t, err)
		})
	}

	err := os.Unsetenv("PATRON_JAEGER_SAMPLER_PARAM")
	assert.NoError(t, err)

	err = os.Unsetenv("PATRON_HTTP_DEFAULT_PORT")
	assert.NoError(t, err)
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

			_, err := New("test", "").WithComponents(tt.cp, tt.cp, tt.cp).build()
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

func (ts testComponent) Run(ctx context.Context) error {
	if ts.errorRunning {
		return errors.New("failed to run component")
	}
	return nil
}

func TestSetupLogging(t *testing.T) {
	t.Run("SetupLogging", func(t *testing.T) {
		err := SetupLogging("myService", "some_version")
		if err != nil {
			t.Errorf("didn't expect an error but got: %s", err)
		}
	})

	t.Run("SetupLoggingWithFields", func(t *testing.T) {
		err := SetupLoggingWithFields("myService", "some_version", map[string]interface{}{"env": "staging"})
		if err != nil {
			t.Errorf("didn't expect an error but got: %s", err)
		}
	})

	t.Run("SetupLoggingWithFields don't override", func(t *testing.T) {
		// this is just to increase coverage
		// we can't assert that the field wasn't overridden as the logger isn't exposed
		err := SetupLoggingWithFields("myService", "some_version", map[string]interface{}{"env": "staging", "srv": "differentService"})
		if err != nil {
			t.Errorf("didn't expect an error but got: %s", err)
		}
	})
}
