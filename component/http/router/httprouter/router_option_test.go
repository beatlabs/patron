package httprouter

import (
	"net/http"
	"testing"

	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/component/http/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRoutes(t *testing.T) {
	t.Parallel()
	type args struct {
		routes []*patronhttp.Route
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{routes: []*patronhttp.Route{{}, {}}}},
		"fail":    {args: args{routes: nil}, expectedErr: "routes are empty"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{}
			err := WithRoutes(tt.args.routes...)(cfg)
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
		acf patronhttp.LivenessCheckFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{acf: func() patronhttp.AliveStatus { return patronhttp.Alive }}},
		"fail":    {args: args{acf: nil}, expectedErr: "alive check function is nil"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{}
			err := WithAliveCheck(tt.args.acf)(cfg)
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
		rcf patronhttp.ReadyCheckFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{rcf: func() patronhttp.ReadyStatus { return patronhttp.Ready }}},
		"fail":    {args: args{rcf: nil}, expectedErr: "ready check function is nil"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{}
			err := WithReadyCheck(tt.args.rcf)(cfg)
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
	type args struct {
		deflateLevel int
	}

	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"too high deflate level":   {args: args{deflateLevel: 10}, expectedErr: "provided deflate level value not in the [-2, 9] range"},
		"too low deflate level":    {args: args{deflateLevel: -3}, expectedErr: "provided deflate level value not in the [-2, 9] range"},
		"acceptable deflate level": {args: args{deflateLevel: 6}},
	}

	for name, tt := range tests {
		temp := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{}
			err := WithDeflateLevel(temp.args.deflateLevel)(cfg)
			if temp.expectedErr != "" {
				assert.EqualError(t, err, temp.expectedErr)
				return
			}

			assert.Equal(t, temp.args.deflateLevel, cfg.deflateLevel)
		})
	}
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
			err := WithMiddlewares(tt.args.mm...)(cfg)
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
	err := WithExpVarProfiling()(cfg)
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
		"missing name":    {args: args{name: "", version: "version"}, expectedErr: "app name cannot be empty"},
		"missing version": {args: args{name: "name", version: ""}, expectedErr: "app version cannot be empty"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			optionFunc, err := WithAppNameHeaders(tt.args.name, tt.args.version)

			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
				assert.Nil(t, optionFunc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, optionFunc)
				cfg := &Config{}
				err = optionFunc(cfg)
				assert.NoError(t, err)
				assert.NotNil(t, optionFunc)
			}
		})
	}
}
