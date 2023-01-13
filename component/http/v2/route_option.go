package v2

import (
	"errors"
	"net/http"

	"github.com/beatlabs/patron/cache"
	"github.com/beatlabs/patron/component/http/auth"
	httpcache "github.com/beatlabs/patron/component/http/cache"
	patronhttp "github.com/beatlabs/patron/component/http/middleware"
	errs "github.com/beatlabs/patron/errors"
	"golang.org/x/time/rate"
)

// WithRateLimiting option for setting a route rate limiter.
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

// WithMiddlewares option for setting the route optionFuncs.
func WithMiddlewares(mm ...patronhttp.Func) RouteOptionFunc {
	return func(r *Route) error {
		if len(mm) == 0 {
			return errors.New("middlewares are empty")
		}
		r.middlewares = append(r.middlewares, mm...)
		return nil
	}
}

// WithAuth option for setting the route auth.
func WithAuth(auth auth.Authenticator) RouteOptionFunc {
	return func(r *Route) error {
		if auth == nil {
			return errors.New("authenticator is nil")
		}
		r.middlewares = append(r.middlewares, patronhttp.NewAuth(auth))
		return nil
	}
}

// WithCache option for setting the route cache.
func WithCache(cache cache.TTLCache, ageBounds httpcache.Age) RouteOptionFunc {
	return func(r *Route) error {
		if r.method != http.MethodGet {
			return errors.New("cannot apply cache to a route with any method other than GET")
		}
		rc, ee := httpcache.NewRouteCache(cache, ageBounds)
		if len(ee) != 0 {
			return errs.Aggregate(ee...)
		}
		m, err := patronhttp.NewCaching(rc)
		if err != nil {
			return errs.Aggregate(err)
		}
		r.middlewares = append(r.middlewares, m)
		return nil
	}
}
