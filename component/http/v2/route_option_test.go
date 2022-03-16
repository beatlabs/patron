package v2

import (
	"net/http"
	"testing"

	"github.com/beatlabs/patron/cache"
	"github.com/beatlabs/patron/cache/redis"
	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/component/http/auth"
	httpcache "github.com/beatlabs/patron/component/http/cache"
	"github.com/stretchr/testify/assert"
)

type MockAuthenticator struct {
	success bool
	err     error
}

func (mo MockAuthenticator) Authenticate(_ *http.Request) (bool, error) {
	if mo.err != nil {
		return false, mo.err
	}
	return mo.success, nil
}

func TestRateLimiting(t *testing.T) {
	t.Parallel()
	route := &Route{}
	assert.NoError(t, RateLimiting(0, 0)(route))
}

func TestRouteMiddlewares(t *testing.T) {
	t.Parallel()
	type args struct {
		mm []patronhttp.MiddlewareFunc
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{mm: []patronhttp.MiddlewareFunc{patronhttp.NewRecoveryMiddleware()}}},
		"fail":    {args: args{mm: nil}, expectedErr: "middlewares are empty"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			route := &Route{}
			err := Middlewares(tt.args.mm...)(route)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.Len(t, route.middlewares, 1)
			}
		})
	}
}

func TestAuth(t *testing.T) {
	t.Parallel()
	type args struct {
		auth auth.Authenticator
	}
	tests := map[string]struct {
		args        args
		expectedErr string
	}{
		"success": {args: args{auth: &MockAuthenticator{}}},
		"fail":    {args: args{auth: nil}, expectedErr: "authenticator is nil"},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			route := &Route{}
			err := Auth(tt.args.auth)(route)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.Len(t, route.middlewares, 1)
			}
		})
	}
}

func TestCache(t *testing.T) {
	t.Parallel()
	type fields struct {
		httpMethod string
	}
	type args struct {
		cache     cache.TTLCache
		ageBounds httpcache.Age
	}
	tests := map[string]struct {
		fields      fields
		args        args
		expectedErr string
	}{
		"success": {
			fields:      fields{httpMethod: http.MethodGet},
			args:        args{cache: &redis.Cache{}, ageBounds: httpcache.Age{}},
			expectedErr: "",
		},
		"fail with missing get": {
			fields:      fields{httpMethod: http.MethodDelete},
			args:        args{cache: &redis.Cache{}, ageBounds: httpcache.Age{}},
			expectedErr: "cannot apply cache to a route with any method other than GET",
		},
		"fail with args": {
			fields:      fields{httpMethod: http.MethodGet},
			args:        args{cache: nil, ageBounds: httpcache.Age{}},
			expectedErr: "route cache is nil\n",
		},
	}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			route := &Route{method: tt.fields.httpMethod}
			err := Cache(tt.args.cache, tt.args.ageBounds)(route)
			if tt.expectedErr != "" {
				assert.EqualError(t, err, tt.expectedErr)
			} else {
				assert.Len(t, route.middlewares, 1)
			}
		})
	}
}
