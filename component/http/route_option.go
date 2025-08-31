package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/beatlabs/patron/cache"
	"github.com/beatlabs/patron/component/http/auth"
	httpcache "github.com/beatlabs/patron/component/http/cache"
	patronhttp "github.com/beatlabs/patron/component/http/middleware"
	"golang.org/x/time/rate"
)

// WithRateLimiting adds a token-bucket rate limiter to the route.
func WithRateLimiting(limit float64, burst int) (RouteOptionFunc, error) {
	if limit < 0 {
		return nil, errors.New("invalid limit")
	}

	if burst < 0 {
		return nil, errors.New("invalid burst")
	}
	m, err := patronhttp.NewRateLimiting(rate.NewLimiter(rate.Limit(limit), burst))
	if err != nil {
		return nil, err
	}

	return func(r *Route) error {
		r.middlewares = append(r.middlewares, m)
		return nil
	}, nil
}

// WithMiddlewares appends middlewares to the route.
func WithMiddlewares(mm ...patronhttp.Func) RouteOptionFunc {
	return func(r *Route) error {
		if len(mm) == 0 {
			return errors.New("middlewares are empty")
		}
		r.middlewares = append(r.middlewares, mm...)
		return nil
	}
}

// WithAuth enforces authentication using the provided authenticator.
func WithAuth(auth auth.Authenticator) RouteOptionFunc {
	return func(r *Route) error {
		if auth == nil {
			return errors.New("authenticator is nil")
		}
		r.middlewares = append(r.middlewares, patronhttp.NewAuth(auth))
		return nil
	}
}

// WithCache enables response caching for GET routes using the provided TTL cache.
func WithCache(cache cache.TTLCache, ageBounds httpcache.Age) RouteOptionFunc {
	return func(r *Route) error {
		if !strings.HasPrefix(r.path, http.MethodGet) {
			return errors.New("cannot apply cache to a route with any method other than GET")
		}
		rc, ee := httpcache.NewRouteCache(cache, ageBounds)
		if len(ee) != 0 {
			return errors.Join(ee...)
		}
		m, err := patronhttp.NewCaching(rc)
		if err != nil {
			return errors.Join(err)
		}
		r.middlewares = append(r.middlewares, m)
		return nil
	}
}
