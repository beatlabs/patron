package patron

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"testing"

	phttp "github.com/beatlabs/patron/sync/http"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	route := phttp.NewRoute("/", "GET", nil, true, nil)
	middleware := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		})
	}
	type args struct {
		name string
		opt  []OptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"success", args{name: "test", opt: []OptionFunc{Routes([]phttp.Route{route}), Middlewares(middleware)}}, false},
		{"failed empty middlewares", args{name: "test", opt: []OptionFunc{Routes([]phttp.Route{route}), Middlewares([]phttp.MiddlewareFunc{}...)}}, true},
		{"failed missing name", args{name: "", opt: []OptionFunc{Routes([]phttp.Route{route})}}, true},
		{"failed missing routes", args{name: "test", opt: []OptionFunc{Routes([]phttp.Route{})}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.name, "", tt.args.opt...)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
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
			s, err := New("test", "", Components(tt.cp, tt.cp, tt.cp))
			assert.NoError(t, err)
			err = s.Run(tt.ctx)
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
			s, err := New("test", "", Components(tt.cp, tt.cp, tt.cp))
			assert.NoError(t, err)
			err = s.Run(tt.ctx)
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
