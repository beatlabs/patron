package http

import (
	"context"
	"net/http"
	"time"

	"github.com/beatlabs/patron/log"
)

// wrapProcessorFunc wraps the processing func with the cache handler interface
func wrapProcessorFunc(path string, processor ProcessorFunc, rc *RouteCache) ProcessorFunc {
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
func wrapHandlerFunc(handler http.HandlerFunc, rc *RouteCache) http.HandlerFunc {
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
