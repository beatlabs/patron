package patron

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/beatlabs/patron/log"
	phttp "github.com/beatlabs/patron/sync/http"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	getRoute := phttp.NewRoute("/", "GET", nil, true, nil)
	putRoute := phttp.NewRoute("/", "PUT", nil, true, nil)

	middleware := phttp.MiddlewareFunc(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	})

	var httpBuilderAllErrors = errors.New("name is required\n" +
		"provided routes slice was empty\n" +
		"provided middlewares slice was empty\n" +
		"alive check func provided was nil\n" +
		"ready check func provided was nil\n" +
		"provided components slice was empty\n" +
		"provided SIGHUP handler was nil\n")

	tests := map[string]struct {
		name          string
		version       string
		cps           []Component
		routes        []phttp.Route
		middlewares   []phttp.MiddlewareFunc
		acf           phttp.AliveCheckFunc
		rcf           phttp.ReadyCheckFunc
		sighupHandler func()
		wantErr       error
	}{
		"success": {
			name:          "test",
			version:       "dev",
			cps:           []Component{&testComponent{}, &testComponent{}},
			routes:        []phttp.Route{getRoute, putRoute},
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
			routes:        nil,
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
			routes:        []phttp.Route{},
			middlewares:   []phttp.MiddlewareFunc{},
			acf:           nil,
			rcf:           nil,
			sighupHandler: nil,
			wantErr:       httpBuilderAllErrors,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotService, gotErr := NewBuilder(tt.name, tt.version).
				WithRoutes(tt.routes).
				WithMiddlewares(tt.middlewares...).
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
				assert.IsType(t, &Service{}, gotService)

				assert.NotEmpty(t, gotService.cps)
				assert.Len(t, gotService.routes, len(tt.routes))
				assert.Len(t, gotService.middlewares, len(tt.middlewares))
				assert.NotNil(t, gotService.rcf)
				assert.NotNil(t, gotService.acf)
				assert.NotNil(t, gotService.termSig)
				assert.NotNil(t, gotService.sighupHandler)

				for _, comp := range tt.cps {
					assert.Contains(t, gotService.cps, comp)
				}
				for i, route := range tt.routes {
					assert.Equal(t, gotService.routes[i].Method, route.Method)
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
			s, err := NewBuilder("test", "").WithComponents(tt.cp, tt.cp, tt.cp).build()
			assert.NoError(t, err)
			err = s.run(tt.ctx)
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
			s, err := NewBuilder("test", "").WithComponents(tt.cp, tt.cp, tt.cp).build()
			assert.NoError(t, err)
			err = s.run(tt.ctx)
			assert.NoError(t, err)
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
