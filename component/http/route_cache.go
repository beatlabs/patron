package http

import (
	"net/http"
	"time"

	"github.com/beatlabs/patron/cache"
)

// RouteCache is the builder needed to build a cache for the corresponding route
type RouteCache struct {
	// cache is the ttl cache implementation to be used
	cache cache.TTLCache
	// age specifies the minimum and maximum amount for max-age and min-fresh Header values respectively
	// regarding the client cache-control requests in seconds
	age age
}

func NewRouteCache(ttlCache cache.TTLCache, age Age) *RouteCache {
	return &RouteCache{
		cache: ttlCache,
		age:   age.toAgeInSeconds(),
	}
}

// Age defines the route cache life-time boundaries for cached objects
type Age struct {
	// Min adds a minimum age threshold for the client controlled cache responses.
	// This will avoid cases where a single client with high request rate and no cache control headers might effectively disable the cache
	// This means that if this parameter is missing (e.g. is equal to '0' , the cache can effectively be made obsolete in the above scenario)
	Min time.Duration
	// Max adds a maximum age for the cache responses. Which effectively works as a time-to-live wrapper on top of the cache
	Max time.Duration
	// The difference of maxAge-minAge sets automatically the max threshold for min-fresh requests
	// This will avoid cases where a single client with high request rate and no cache control headers might effectively disable the cache
	// This means that if this parameter is very high (e.g. greater than ttl , the cache can effectively be made obsolete in the above scenario)
}

func (a Age) toAgeInSeconds() age {
	return age{
		min: int64(a.Min / time.Second),
		max: int64(a.Max / time.Second),
	}
}

type age struct {
	min int64
	max int64
}

// handlerExecutor is the function that will create a new CachedResponse based on a HandlerFunc implementation
func handlerExecutor(_ http.ResponseWriter, request *http.Request, hnd http.HandlerFunc) executor {
	return func(now int64, key string) *CachedResponse {
		var err error
		responseReadWriter := newResponseReadWriter()
		hnd(responseReadWriter, request)
		payload, err := responseReadWriter.readAll()
		rw := *responseReadWriter
		if err == nil {
			return &CachedResponse{
				Response: CacheHandlerResponse{
					Bytes: payload,
					// cache also the headers generated by the handler
					Header: extractHeaders(rw.Header()),
				},
				LastValid: now,
				Etag:      generateETag([]byte(key), time.Now().Nanosecond()),
			}
		}
		return &CachedResponse{Err: err}
	}
}
