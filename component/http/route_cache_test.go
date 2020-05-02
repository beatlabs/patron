package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/beatlabs/patron/cache"
	httppatron "github.com/beatlabs/patron/client/http"

	"github.com/stretchr/testify/assert"
)

type cacheState struct {
	setOps int
	getOps int
	size   int
}

type builderOperation func(routeBuilder *RouteBuilder) *RouteBuilder

type arg struct {
	bop builderOperation
	age Age
	err bool
}

func TestNewRouteBuilder_WithCache(t *testing.T) {

	args := []arg{
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodGet()
			},
			age: Age{Max: 10},
		},
		// error with '0' ttl
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodGet()
			},
			age: Age{Min: 10, Max: 1},
			err: true,
		},
		// error for POST method
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodPost()
			},
			age: Age{Max: 10},
			err: true,
		},
	}

	c := newTestingCache()

	processor := func(context context.Context, request *Request) (response *Response, e error) {
		return nil, nil
	}

	handler := func(writer http.ResponseWriter, i *http.Request) {
	}

	for _, arg := range args {

		assertRouteBuilder(t, arg, NewRouteBuilder("/", processor), c)

		assertRouteBuilder(t, arg, NewRawRouteBuilder("/", handler), c)

	}
}

func assertRouteBuilder(t *testing.T, arg arg, routeBuilder *RouteBuilder, cache cache.TTLCache) {

	routeBuilder.WithRouteCache(cache, arg.age)

	if arg.bop != nil {
		routeBuilder = arg.bop(routeBuilder)
	}

	route, err := routeBuilder.Build()
	assert.NotNil(t, route)

	if arg.err {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}
}

func TestRouteCacheImplementation_WithSingleRequest(t *testing.T) {

	cc := newTestingCache()
	cc.instant = now

	var executions uint32

	preWrapper := newMiddlewareWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("pre-middleware-header", "pre")
	})

	postWrapper := newMiddlewareWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("post-middleware-header", "post")
	})

	routeBuilder := NewRouteBuilder("/path", func(context context.Context, request *Request) (response *Response, e error) {
		atomic.AddUint32(&executions, 1)
		newResponse := NewResponse("body")
		newResponse.Header["Custom-Header"] = "11"
		return newResponse, nil
	}).
		WithRouteCache(cc, Age{Max: 10 * time.Second}).
		WithMiddlewares(preWrapper.middleware, postWrapper.middleware).
		MethodGet()

	ctx, cln := context.WithTimeout(context.Background(), 5*time.Second)

	port := 50023
	runRoute(ctx, t, routeBuilder, port)

	assertResponse(ctx, t, []http.Response{
		{
			Header: map[string][]string{
				cacheControlHeader:       {"max-age=10"},
				"Content-Type":           {"application/json; charset=utf-8"},
				"Content-Length":         {"6"},
				"Post-Middleware-Header": {"post"},
				"Pre-Middleware-Header":  {"pre"},
				"Custom-Header":          {"11"},
			},
			Body: &bodyReader{body: "\"body\""},
		},
		{
			Header: map[string][]string{
				cacheControlHeader:       {"max-age=10"},
				"Content-Type":           {"application/json; charset=utf-8"},
				"Content-Length":         {"6"},
				"Post-Middleware-Header": {"post"},
				"Pre-Middleware-Header":  {"pre"},
				"Custom-Header":          {"11"},
			},
			Body: &bodyReader{body: "\"body\""},
		},
	}, port)

	assertCacheState(t, *cc, cacheState{
		setOps: 1,
		getOps: 2,
		size:   1,
	})

	assert.Equal(t, 2, preWrapper.invocations)
	assert.Equal(t, 2, postWrapper.invocations)

	assert.Equal(t, executions, uint32(1))

	cln()
}

func TestRouteCacheAsMiddleware_WithSingleRequest(t *testing.T) {

	cc := newTestingCache()
	cc.instant = now

	var executions uint32

	preWrapper := newMiddlewareWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("pre-middleware-header", "pre")
	})

	postWrapper := newMiddlewareWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("post-middleware-header", "post")
	})

	routeBuilder := NewRouteBuilder("/path", func(context context.Context, request *Request) (response *Response, e error) {
		atomic.AddUint32(&executions, 1)
		newResponse := NewResponse("body")
		newResponse.Header["internal-handler-header"] = "header"
		return newResponse, nil
	}).
		WithMiddlewares(
			preWrapper.middleware,
			NewCachingMiddleware(NewRouteCache(cc, Age{Max: 10 * time.Second})),
			postWrapper.middleware).
		MethodGet()

	ctx, cln := context.WithTimeout(context.Background(), 5*time.Second)

	port := 50023
	runRoute(ctx, t, routeBuilder, port)

	assertResponse(ctx, t, []http.Response{
		{
			Header: map[string][]string{
				cacheControlHeader:        {"max-age=10"},
				"Content-Type":            {"application/json; charset=utf-8"},
				"Content-Length":          {"6"},
				"Post-Middleware-Header":  {"post"},
				"Pre-Middleware-Header":   {"pre"},
				"Internal-Handler-Header": {"header"},
			},
			Body: &bodyReader{body: "\"body\""},
		},
		{
			Header: map[string][]string{
				cacheControlHeader:        {"max-age=10"},
				"Content-Type":            {"application/json; charset=utf-8"},
				"Post-Middleware-Header":  {"post"},
				"Pre-Middleware-Header":   {"pre"},
				"Content-Length":          {"6"},
				"Internal-Handler-Header": {"header"},
			},
			Body: &bodyReader{body: "\"body\""},
		},
	}, port)

	assertCacheState(t, *cc, cacheState{
		setOps: 1,
		getOps: 2,
		size:   1,
	})

	assert.Equal(t, 2, preWrapper.invocations)
	// NOTE : the post middleware is not executed, as it s hidden behind the cache
	assert.Equal(t, 1, postWrapper.invocations)

	assert.Equal(t, executions, uint32(1))

	cln()

}

type middlewareWrapper struct {
	middleware  MiddlewareFunc
	invocations int
}

func newMiddlewareWrapper(middlewareFunc func(w http.ResponseWriter, r *http.Request)) *middlewareWrapper {
	wrapper := &middlewareWrapper{}
	wrapper.middleware = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrapper.invocations++
			middlewareFunc(w, r)
			next.ServeHTTP(w, r)
		})
	}
	return wrapper
}

func TestRawRouteCacheImplementation_WithSingleRequest(t *testing.T) {

	cc := newTestingCache()
	cc.instant = now

	var executions uint32

	preWrapper := newMiddlewareWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("pre-middleware-header", "pre")
	})

	postWrapper := newMiddlewareWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("post-middleware-header", "post")
	})

	routeBuilder := NewRawRouteBuilder("/path", func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddUint32(&executions, 1)
		i, err := writer.Write([]byte("\"body\""))
		writer.Header().Set("internal-handler-header", "header")
		assert.NoError(t, err)
		assert.True(t, i > 0)
	}).
		WithRouteCache(cc, Age{Max: 10 * time.Second}).
		WithMiddlewares(preWrapper.middleware, postWrapper.middleware).
		MethodGet()

	ctx, cln := context.WithTimeout(context.Background(), 5*time.Second)

	port := 50024
	runRoute(ctx, t, routeBuilder, port)

	assertResponse(ctx, t, []http.Response{
		{
			Header: map[string][]string{
				cacheControlHeader:        {"max-age=10"},
				"Content-Type":            {"text/plain; charset=utf-8"},
				"Content-Length":          {"6"},
				"Post-Middleware-Header":  {"post"},
				"Pre-Middleware-Header":   {"pre"},
				"Internal-Handler-Header": {"header"}},
			Body: &bodyReader{body: "\"body\""},
		},
		{
			Header: map[string][]string{
				cacheControlHeader:        {"max-age=10"},
				"Content-Type":            {"text/plain; charset=utf-8"},
				"Content-Length":          {"6"},
				"Post-Middleware-Header":  {"post"},
				"Pre-Middleware-Header":   {"pre"},
				"Internal-Handler-Header": {"header"}},
			Body: &bodyReader{body: "\"body\""},
		},
	}, port)

	assertCacheState(t, *cc, cacheState{
		setOps: 1,
		getOps: 2,
		size:   1,
	})

	assert.Equal(t, 2, preWrapper.invocations)
	assert.Equal(t, 2, postWrapper.invocations)

	assert.Equal(t, executions, uint32(1))

	cln()

}

type bodyReader struct {
	body string
}

func (br *bodyReader) Read(p []byte) (n int, err error) {
	var c int
	for i, b := range []byte(br.body) {
		p[i] = b
		c = i
	}
	return c + 1, nil
}

func (br *bodyReader) Close() error {
	// nothing to do
	return nil
}

func runRoute(ctx context.Context, t *testing.T, routeBuilder *RouteBuilder, port int) {
	cmp, err := NewBuilder().WithRoutesBuilder(NewRoutesBuilder().Append(routeBuilder)).WithPort(port).Create()

	assert.NoError(t, err)
	assert.NotNil(t, cmp)

	go func() {
		err = cmp.Run(ctx)
		assert.NoError(t, err)
	}()

	var lwg sync.WaitGroup
	lwg.Add(1)
	go func() {
		cl, err := httppatron.New()
		assert.NoError(t, err)
		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/ready", port), nil)
		assert.NoError(t, err)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				r, err := cl.Do(ctx, req)
				if err == nil && r != nil {
					lwg.Done()
					return
				}
			}
		}
	}()
	lwg.Wait()
}

func assertResponse(ctx context.Context, t *testing.T, expected []http.Response, port int) {

	cl, err := httppatron.New()
	assert.NoError(t, err)
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/path", port), nil)
	assert.NoError(t, err)

	for _, expectedResponse := range expected {
		response, err := cl.Do(ctx, req)

		assert.NoError(t, err)

		for k, v := range expectedResponse.Header {
			assert.Equal(t, v, response.Header[k])
		}
		assert.Equal(t, expectedResponse.Header.Get(cacheControlHeader), response.Header.Get(cacheControlHeader))
		assert.True(t, response.Header.Get(cacheHeaderETagHeader) != "")
		expectedPayload := make([]byte, 6)
		i, err := expectedResponse.Body.Read(expectedPayload)
		assert.NoError(t, err)

		responsePayload := make([]byte, 6)
		j, err := response.Body.Read(responsePayload)
		assert.Error(t, err)

		assert.Equal(t, i, j)
		assert.Equal(t, expectedPayload, responsePayload)
	}

}

func assertCacheState(t *testing.T, cache testingCache, cacheState cacheState) {
	assert.Equal(t, cacheState.setOps, cache.setCount)
	assert.Equal(t, cacheState.getOps, cache.getCount)
	assert.Equal(t, cacheState.size, cache.size())
}
