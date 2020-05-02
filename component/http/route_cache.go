package http

import (
	"context"
	"net/http"
	"time"

	"github.com/beatlabs/patron/cache"

	"github.com/beatlabs/patron/log"
)

// routeCache is the builder needed to build a cache for the corresponding route
type routeCache struct {
	// cache is the ttl cache implementation to be used
	cache cache.TTLCache
	// age specifies the minimum and maximum amount for max-age and min-fresh header values respectively
	// regarding the client cache-control requests in seconds
	age age
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

// wrapProcessorFunc wraps the processing func with the cache handler interface
func wrapProcessorFunc(path string, processor ProcessorFunc, rc *routeCache) ProcessorFunc {
	return func(ctx context.Context, request *Request) (response *Response, e error) {
		// we are doing the opposite work that we would do in the processor,
		// but until we refactor this part this seems the only way
		req := fromRequest(path, request)
		resp, err := cacheHandler(processorExecutor(ctx, request, processor), rc)(req)
		if err != nil {
			return nil, err
		}
		return &Response{Payload: resp.payload, Headers: resp.header}, nil
	}
}

// processorExecutor is the function that will create a new cachedResponse based on a ProcessorFunc implementation
func processorExecutor(ctx context.Context, request *Request, hnd ProcessorFunc) executor {
	return func(now int64, key string) *cachedResponse {
		var err error
		response, err := hnd(ctx, request)
		if err == nil {
			return &cachedResponse{
				response: &cacheHandlerResponse{
					payload: response.Payload,
					header:  make(map[string]string),
				},
				lastValid: now,
				etag:      generateETag([]byte(key), time.Now().Nanosecond()),
			}
		}
		return &cachedResponse{err: err}
	}
}

// wrapHandlerFunc wraps the handler func with the cache handler interface
func wrapHandlerFunc(handler http.HandlerFunc, rc *routeCache) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		req := fromHTTPRequest(request)
		if resp, err := cacheHandler(handlerExecutor(response, request, handler), rc)(req); err != nil {
			log.Errorf("could not handle request with the cache processor: %v", err)
		} else {
			propagateHeaders(resp.header, response.Header())
			if i, err := response.Write(resp.bytes); err != nil {
				log.Errorf("could not Write cache processor result into response %d: %v", i, err)
			}
		}
	}
}

// handlerExecutor is the function that will create a new cachedResponse based on a HandlerFunc implementation
func handlerExecutor(_ http.ResponseWriter, request *http.Request, hnd http.HandlerFunc) executor {
	return func(now int64, key string) *cachedResponse {
		var err error
		responseReadWriter := newResponseReadWriter()
		hnd(responseReadWriter, request)
		payload, err := responseReadWriter.readAll()
		if err == nil {
			return &cachedResponse{
				response: &cacheHandlerResponse{
					bytes:  payload,
					header: make(map[string]string),
				},
				lastValid: now,
				etag:      generateETag([]byte(key), time.Now().Nanosecond()),
			}
		}
		return &cachedResponse{err: err}
	}
}
