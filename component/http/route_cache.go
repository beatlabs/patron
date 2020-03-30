package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/beatlabs/patron/log"
)

func wrapProcessorFunc(path string, processor ProcessorFunc, rc *routeCache) ProcessorFunc {
	return func(ctx context.Context, request *Request) (response *Response, e error) {
		// we are doing the opposite work that we would do in the processor,
		// but until we refactor this part this seems the only way
		req := &cacheHandlerRequest{}
		req.fromRequest(path, request)
		resp, err := cacheHandler(processorExecutor(ctx, request, processor), rc)(req)
		if err != nil {
			return nil, err
		}
		return &Response{Payload: resp.payload, Headers: resp.header}, nil
	}
}

// processorExecutor is the function that will create a new cachedResponse based on a ProcessorFunc implementation
var processorExecutor = func(ctx context.Context, request *Request, hnd ProcessorFunc) executor {
	return func(now int64, key string) *cachedResponse {
		if response, err := hnd(ctx, request); err == nil {
			return &cachedResponse{
				response: &cacheHandlerResponse{
					payload: response.Payload,
					header:  make(map[string]string),
				},
				lastValid: now,
				etag:      generateETag([]byte(key), time.Now().Nanosecond()),
			}
		} else {
			return &cachedResponse{err: err}
		}
	}
}

func wrapHandlerFunc(handler http.HandlerFunc, rc *routeCache) http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		req := &cacheHandlerRequest{}
		req.fromHTTPRequest(request)
		if resp, err := cacheHandler(handlerExecutor(response, request, handler), rc)(req); err != nil {
			log.Errorf("could not handle request with the cache processor: %v", err)
		} else {
			println(fmt.Sprintf("resp = %v", resp))
			propagateHeaders(resp.header, response.Header())
			if i, err := response.Write(resp.bytes); err != nil {
				log.Errorf("could not write cache processor result into response %d: %v", i, err)
			}
		}
	}
}

// handlerExecutor is the function that will create a new cachedResponse based on a HandlerFunc implementation
var handlerExecutor = func(response http.ResponseWriter, request *http.Request, hnd http.HandlerFunc) executor {
	return func(now int64, key string) *cachedResponse {
		responseReadWriter := NewResponseReadWriter()
		hnd(responseReadWriter, request)
		if payload, err := responseReadWriter.ReadAll(); err == nil {
			return &cachedResponse{
				response: &cacheHandlerResponse{
					bytes:  payload,
					header: make(map[string]string),
				},
				lastValid: now,
				etag:      generateETag([]byte(key), time.Now().Nanosecond()),
			}
		} else {
			return &cachedResponse{err: err}
		}

	}
}
