package http

import (
	"errors"
	"fmt"
	"time"

	"github.com/beatlabs/patron/cache"
	errs "github.com/beatlabs/patron/errors"
)

// RouteCacheBuilder is the builder needed to build a cache for the corresponding route
type RouteCacheBuilder struct {
	cache         cache.Cache
	instant       TimeInstant
	ttl           time.Duration
	minAge        time.Duration
	maxFresh      time.Duration
	staleResponse bool
	errors        []error
}

// NewRouteCacheBuilder creates a new builder for the route cache implementation.
func NewRouteCacheBuilder(cache cache.Cache, ttl time.Duration) *RouteCacheBuilder {

	var ee []error

	if ttl <= 0 {
		ee = append(ee, errors.New("time to live must be greater than `0`"))
	}

	return &RouteCacheBuilder{
		cache: cache,
		ttl:   ttl,
		instant: func() int64 {
			return time.Now().Unix()
		},
		errors: ee,
	}
}

// WithTimeInstant specifies a time instant function for checking expiry.
func (cb *RouteCacheBuilder) WithTimeInstant(instant TimeInstant) *RouteCacheBuilder {
	if instant == nil {
		cb.errors = append(cb.errors, errors.New("time instant is nil"))
	}
	cb.instant = instant
	return cb
}

// WithMinAge adds a minimum age for the cache responses.
// This will avoid cases where a single client with high request rate and no cache control headers might effectively disable the cache
// This means that if this parameter is missing (e.g. is equal to '0' , the cache can effectively be made obsolete in the above scenario)
func (cb *RouteCacheBuilder) WithMinAge(minAge time.Duration) *RouteCacheBuilder {
	if minAge <= 0 {
		cb.errors = append(cb.errors, fmt.Errorf("min-age cannot be lower or equal to zero '0 < %v'", minAge))
	}
	cb.minAge = minAge
	return cb
}

// WithMaxFresh adds a maximum age for the cache responses.
// This will avoid cases where a single client with high request rate and no cache control headers might effectively disable the cache
// This means that if this parameter is very high (e.g. greater than ttl , the cache can effectively be made obsolete in the above scenario)
func (cb *RouteCacheBuilder) WithMaxFresh(maxFresh time.Duration) *RouteCacheBuilder {
	if maxFresh > cb.ttl {
		cb.errors = append(cb.errors, fmt.Errorf("max-fresh cannot be greater than the time to live '%v <= %v'", maxFresh, cb.ttl))
	}
	cb.maxFresh = maxFresh
	return cb
}

// WithStaleResponse allows the cache to return stale responses.
func (cb *RouteCacheBuilder) WithStaleResponse(staleResponse bool) *RouteCacheBuilder {
	cb.staleResponse = staleResponse
	return cb
}

func (cb *RouteCacheBuilder) create() (*routeCache, error) {
	if len(cb.errors) > 0 {
		return nil, errs.Aggregate(cb.errors...)
	}

	if cb.maxFresh == 0 {
		cb.maxFresh = cb.ttl
	}

	if cb.minAge == 0 {
		cb.minAge = cb.ttl
	}

	return &routeCache{
		cache:         cb.cache,
		ttl:           int64(cb.ttl / time.Second),
		instant:       cb.instant,
		minAge:        int64(cb.minAge / time.Second),
		maxFresh:      int64(cb.maxFresh / time.Second),
		staleResponse: cb.staleResponse,
	}, nil
}

type routeCache struct {
	// cache is the cache implementation to be used
	cache cache.Cache
	// ttl is the time to live for all cached objects in seconds
	ttl int64
	// instant is the timing function for the cache expiry calculations
	instant TimeInstant
	// minAge specifies the minimum amount of max-age header value for client cache-control requests in seconds
	minAge int64
	// max-fresh specifies the maximum amount of min-fresh header value for client cache-control requests in seconds
	maxFresh int64
	// staleResponse specifies if the server is willing to send stale responses
	// if a new response could not be generated for any reason
	staleResponse bool
}
