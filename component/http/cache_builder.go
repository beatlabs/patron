package http

import (
	"errors"
	"time"

	"github.com/beatlabs/patron/cache"
)

// RouteCache is the builder needed to build a cache for the corresponding route
type RouteCache struct {
	// cache is the cache implementation to be used
	cache cache.Cache
	// instant is the timing function for the cache expiry calculations
	instant TimeInstant
	// age specifies the minimum and maximum amount for max-age and min-fresh header values respectively
	// regarding the client cache-control requests in seconds
	age age
	// staleResponse specifies if the server is willing to send stale responses
	// if a new response could not be generated for any reason
	staleResponse bool
	errors        []error
}

type Age struct {
	// Min adds a minimum age for the cache responses.
	// This will avoid cases where a single client with high request rate and no cache control headers might effectively disable the cache
	// This means that if this parameter is missing (e.g. is equal to '0' , the cache can effectively be made obsolete in the above scenario)
	Min time.Duration
	// Max adds a maximum age for the cache responses. Which effectively works as a time-to-live wrapper on top of the cache
	Max time.Duration
	// The difference of maxAge-minAge sets automatically the max threshold for min-fresh requests
	// This will avoid cases where a single client with high request rate and no cache control headers might effectively disable the cache
	// This means that if this parameter is very high (e.g. greater than ttl , the cache can effectively be made obsolete in the above scenario)
}

type age struct {
	min int64
	max int64
}

// NewRouteCache creates a new builder for the route cache implementation.
func NewRouteCache(cache cache.Cache, ageBounds Age) *RouteCache {

	var ee []error

	if ageBounds.Max <= 0 {
		ee = append(ee, errors.New("max age must be greater than `0`"))
	}

	return &RouteCache{
		cache: cache,
		instant: func() int64 {
			return time.Now().Unix()
		},
		age: age{
			min: int64(ageBounds.Min / time.Second),
			max: int64(ageBounds.Max / time.Second),
		},
		errors: ee,
	}
}

// WithTimeInstant specifies a time instant function for checking expiry.
func (cb *RouteCache) WithTimeInstant(instant TimeInstant) *RouteCache {
	if instant == nil {
		cb.errors = append(cb.errors, errors.New("time instant is nil"))
	}
	cb.instant = instant
	return cb
}

// WithStaleResponse allows the cache to return stale responses.
func (cb *RouteCache) WithStaleResponse(staleResponse bool) *RouteCache {
	cb.staleResponse = staleResponse
	return cb
}
