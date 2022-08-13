package httprouter

import (
	"net/http"
	"testing"

	"github.com/beatlabs/patron/component/http/middleware"
	v2 "github.com/beatlabs/patron/component/http/v2"
	"github.com/stretchr/testify/assert"
)

func TestRoutes(t *testing.T) {
	t.Parallel()
	type args struct {
		routes []*v2.Route
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{routes: []*v2.Route{{}, {}}}},
		"fail":    {args: args{routes: nil}, expectedErr: "routes are empty"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{}
			err := Routes(tt.args.routes...)(cfg)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.Equal(t, tt.args.routes, cfg.routes)
			}
		})
	}
}

func TestAliveCheck(t *testing.T) {
	t.Parallel()
	type args struct {
		acf v2.LivenessCheckFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{acf: func() v2.AliveStatus { return v2.Alive }}},
		"fail":    {args: args{acf: nil}, expectedErr: "alive check function is nil"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{}
			err := AliveCheck(tt.args.acf)(cfg)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NotNil(t, cfg.aliveCheckFunc)
			}
		})
	}
}

func TestReadyCheck(t *testing.T) {
	t.Parallel()
	type args struct {
		rcf v2.ReadyCheckFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{rcf: func() v2.ReadyStatus { return v2.Ready }}},
		"fail":    {args: args{rcf: nil}, expectedErr: "ready check function is nil"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{}
			err := ReadyCheck(tt.args.rcf)(cfg)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.NotNil(t, cfg.readyCheckFunc)
			}
		})
	}
}

func TestDeflateLevel(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	err := DeflateLevel(10)(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 10, cfg.deflateLevel)
}

func TestMiddlewares(t *testing.T) {
	t.Parallel()
	type args struct {
		mm []middleware.Func
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{mm: []middleware.Func{func(next http.Handler) http.Handler { return next }}}},
		"fail":    {args: args{mm: nil}, expectedErr: "middlewares are empty"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{}
			err := Middlewares(tt.args.mm...)(cfg)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.Len(t, cfg.middlewares, 1)
			}
		})
	}
}

func TestDisableProfiling(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	err := EnableExpVarProfiling()(cfg)
	assert.NoError(t, err)
	assert.True(t, cfg.enableProfilingExpVar)
}

func TestEnableAppNameHeaders(t *testing.T) {
	type args struct {
		name    string
		version string
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success":         {args: args{name: "name", version: "version"}},
		"missing name":    {args: args{name: "", version: "version"}, expectedErr: "app name was not provided"},
		"missing version": {args: args{name: "name", version: ""}, expectedErr: "app version was not provided"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			cfg := &Config{}
			err := EnableAppNameHeaders(tt.args.name, tt.args.version)(cfg)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, cfg.appNameVersionMiddleware)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg.appNameVersionMiddleware)
			}
		})
	}
}
