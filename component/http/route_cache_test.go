package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/beatlabs/patron/encoding"

	"github.com/beatlabs/patron/cache"
	httpclient "github.com/beatlabs/patron/client/http"
	httpcache "github.com/beatlabs/patron/component/http/cache"

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
	age httpcache.Age
	err bool
}

func TestCachingMiddleware(t *testing.T) {
	getRequest, err := http.NewRequest("GET", "/test", nil)
	assert.NoError(t, err)

	postRequest, err := http.NewRequest("POST", "/test", nil)
	assert.NoError(t, err)

	type args struct {
		next http.Handler
		mws  []MiddlewareFunc
	}

	testingCache := newTestingCache()
	testingCache.instant = httpcache.NowSeconds

	routeCache, errs := httpcache.NewRouteCache(testingCache, httpcache.Age{Max: 1 * time.Second})
	assert.Empty(t, errs)

	tests := []struct {
		name         string
		args         args
		r            *http.Request
		expectedCode int
		expectedBody string
		cacheState   cacheState
	}{
		{
			"caching middleware with POST request",
			args{next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(202)
				i, err := w.Write([]byte{1, 2, 3, 4})
				assert.NoError(t, err)
				assert.Equal(t, 4, i)
			}), mws: []MiddlewareFunc{NewCachingMiddleware(routeCache)}},
			postRequest, 202, "\x01\x02\x03\x04",
			cacheState{},
		},
		{
			"caching middleware with GET request",
			args{next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				i, err := w.Write([]byte{1, 2, 3, 4})
				assert.NoError(t, err)
				assert.Equal(t, 4, i)
			}), mws: []MiddlewareFunc{NewCachingMiddleware(routeCache)}},
			getRequest, 200, "\x01\x02\x03\x04",
			cacheState{
				setOps: 1,
				getOps: 1,
				size:   1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := httptest.NewRecorder()
			rw := newResponseWriter(rc)
			tt.args.next = MiddlewareChain(tt.args.next, tt.args.mws...)
			tt.args.next.ServeHTTP(rw, tt.r)
			assert.Equal(t, tt.expectedCode, rw.Status())
			assert.Equal(t, tt.expectedBody, rc.Body.String())
			assertCacheState(t, *testingCache, tt.cacheState)
		})
	}
}

func TestNewRouteBuilder_WithCache(t *testing.T) {
	args := []arg{
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodGet()
			},
			age: httpcache.Age{Max: 10},
		},
		// error with '0' ttl
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodGet()
			},
			age: httpcache.Age{Min: 10, Max: 1},
			err: true,
		},
		// error for POST method
		{
			bop: func(routeBuilder *RouteBuilder) *RouteBuilder {
				return routeBuilder.MethodPost()
			},
			age: httpcache.Age{Max: 10},
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
	ce := make(chan error, 1)

	cc := newTestingCache()
	cc.instant = httpcache.NowSeconds

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
		WithRouteCache(cc, httpcache.Age{Max: 10 * time.Second}).
		WithMiddlewares(preWrapper.middleware, postWrapper.middleware).
		MethodGet()

	ctx, cln := context.WithTimeout(context.Background(), 5*time.Second)

	port := 50023
	runRoute(ctx, t, routeBuilder, ce, port)

	assertResponse(ctx, t, []http.Response{
		{
			Header: map[string][]string{
				httpcache.HeaderCacheControl: {"max-age=10"},
				"Content-Type":               {"application/json; charset=utf-8"},
				"Content-Length":             {"6"},
				"Post-Middleware-Header":     {"post"},
				"Pre-Middleware-Header":      {"pre"},
				"Custom-Header":              {"11"},
			},
			Body: &bodyReader{body: "\"body\""},
		},
		{
			Header: map[string][]string{
				httpcache.HeaderCacheControl: {"max-age=10"},
				"Content-Type":               {"application/json; charset=utf-8"},
				"Content-Length":             {"6"},
				"Post-Middleware-Header":     {"post"},
				"Pre-Middleware-Header":      {"pre"},
				"Custom-Header":              {"11"},
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
	assert.NoError(t, <-ce)
}

func TestRouteCacheAsMiddleware_WithSingleRequest(t *testing.T) {
	ce := make(chan error, 1)

	cc := newTestingCache()
	cc.instant = httpcache.NowSeconds

	var executions uint32

	preWrapper := newMiddlewareWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("pre-middleware-header", "pre")
	})

	postWrapper := newMiddlewareWrapper(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("post-middleware-header", "post")
	})

	routeCache, errs := httpcache.NewRouteCache(cc, httpcache.Age{Max: 10 * time.Second})
	assert.Empty(t, errs)
	routeBuilder := NewRouteBuilder("/path", func(context context.Context, request *Request) (response *Response, e error) {
		atomic.AddUint32(&executions, 1)
		newResponse := NewResponse("body")
		newResponse.Header["internal-handler-header"] = "header"
		return newResponse, nil
	}).
		WithMiddlewares(
			preWrapper.middleware,
			NewCachingMiddleware(routeCache),
			postWrapper.middleware).
		MethodGet()

	ctx, cln := context.WithTimeout(context.Background(), 5*time.Second)

	port := 50023
	runRoute(ctx, t, routeBuilder, ce, port)

	assertResponse(ctx, t, []http.Response{
		{
			Header: map[string][]string{
				httpcache.HeaderCacheControl: {"max-age=10"},
				"Content-Type":               {"application/json; charset=utf-8"},
				"Content-Length":             {"6"},
				"Post-Middleware-Header":     {"post"},
				"Pre-Middleware-Header":      {"pre"},
				"Internal-Handler-Header":    {"header"},
			},
			Body: &bodyReader{body: "\"body\""},
		},
		{
			Header: map[string][]string{
				httpcache.HeaderCacheControl: {"max-age=10"},
				"Content-Type":               {"application/json; charset=utf-8"},
				"Post-Middleware-Header":     {"post"},
				"Pre-Middleware-Header":      {"pre"},
				"Content-Length":             {"6"},
				"Internal-Handler-Header":    {"header"},
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
	assert.NoError(t, <-ce)
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
	ce := make(chan error, 1)

	cc := newTestingCache()
	cc.instant = httpcache.NowSeconds

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
		WithRouteCache(cc, httpcache.Age{Max: 10 * time.Second}).
		WithMiddlewares(preWrapper.middleware, postWrapper.middleware).
		MethodGet()

	ctx, cln := context.WithTimeout(context.Background(), 5*time.Second)

	port := 50024
	runRoute(ctx, t, routeBuilder, ce, port)

	assertResponse(ctx, t, []http.Response{
		{
			Header: map[string][]string{
				httpcache.HeaderCacheControl: {"max-age=10"},
				"Content-Type":               {"text/plain; charset=utf-8"},
				"Content-Length":             {"6"},
				"Post-Middleware-Header":     {"post"},
				"Pre-Middleware-Header":      {"pre"},
				"Internal-Handler-Header":    {"header"},
			},
			Body: &bodyReader{body: "\"body\""},
		},
		{
			Header: map[string][]string{
				httpcache.HeaderCacheControl: {"max-age=10"},
				"Content-Type":               {"text/plain; charset=utf-8"},
				"Content-Length":             {"6"},
				"Post-Middleware-Header":     {"post"},
				"Pre-Middleware-Header":      {"pre"},
				"Internal-Handler-Header":    {"header"},
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
	assert.NoError(t, <-ce)
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

func runRoute(ctx context.Context, t *testing.T, routeBuilder *RouteBuilder, ce chan error, port int) {
	cmp, err := NewBuilder().WithRoutesBuilder(NewRoutesBuilder().Append(routeBuilder)).WithPort(port).Create()

	assert.NoError(t, err)
	assert.NotNil(t, cmp)

	go func() {
		ce <- cmp.Run(ctx)
		close(ce)
	}()

	var lwg sync.WaitGroup
	lwg.Add(1)
	go func() {
		cl, err := httpclient.New()
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
	cl, err := httpclient.New()
	assert.NoError(t, err)
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/path", port), nil)
	require.NoError(t, err)
	req.Header.Set(encoding.AcceptEncodingHeader, "*") // bypass default HTTP client GZIP-ing our response
	assert.NoError(t, err)

	for _, expectedResponse := range expected {
		response, err := cl.Do(ctx, req)

		assert.NoError(t, err)

		for k, v := range expectedResponse.Header {
			assert.Equal(t, v, response.Header[k])
		}
		assert.Equal(t, expectedResponse.Header.Get(httpcache.HeaderCacheControl), response.Header.Get(httpcache.HeaderCacheControl))
		assert.True(t, response.Header.Get(httpcache.HeaderETagHeader) != "")
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

type testingCacheEntity struct {
	v   interface{}
	ttl int64
	t0  int64
}

type testingCache struct {
	cache    map[string]testingCacheEntity
	getCount int
	setCount int
	getErr   error
	setErr   error
	instant  func() int64
}

func newTestingCache() *testingCache {
	return &testingCache{cache: make(map[string]testingCacheEntity)}
}

func (t *testingCache) Get(key string) (interface{}, bool, error) {
	t.getCount++
	if t.getErr != nil {
		return nil, false, t.getErr
	}
	r, ok := t.cache[key]
	if t.instant()-r.t0 > r.ttl {
		return nil, false, nil
	}
	return r.v, ok, nil
}

func (t *testingCache) Purge() error {
	for k := range t.cache {
		_ = t.Remove(k)
	}
	return nil
}

func (t *testingCache) Remove(key string) error {
	delete(t.cache, key)
	return nil
}

// Note : this method will effectively not cache anything
// e.g. testingCacheEntity.t is `0`
func (t *testingCache) Set(key string, value interface{}) error {
	t.setCount++
	if t.setErr != nil {
		return t.getErr
	}
	t.cache[key] = testingCacheEntity{
		v: value,
	}
	return nil
}

func (t *testingCache) SetTTL(key string, value interface{}, ttl time.Duration) error {
	t.setCount++
	if t.setErr != nil {
		return t.getErr
	}
	t.cache[key] = testingCacheEntity{
		v:   value,
		ttl: int64(ttl / time.Second),
		t0:  t.instant(),
	}
	return nil
}

func (t *testingCache) size() int {
	return len(t.cache)
}
