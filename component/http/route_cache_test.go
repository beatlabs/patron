package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	client "github.com/beatlabs/patron/client/http"

	"github.com/stretchr/testify/assert"
)

type cacheState struct {
	setOps int
	getOps int
	size   int
}

func TestProcessorWrapper(t *testing.T) {

	type arg struct {
		processor ProcessorFunc
		req       *Request
		err       bool
	}

	args := []arg{
		{
			processor: func(ctx context.Context, request *Request) (response *Response, e error) {
				return nil, errors.New("processor error")
			},
			req: NewRequest(make(map[string]string), nil, make(map[string]string), nil),
			err: true,
		},
		{
			processor: func(ctx context.Context, request *Request) (response *Response, e error) {
				return NewResponse(request.Fields), nil
			},
			req: NewRequest(make(map[string]string), nil, make(map[string]string), nil),
		},
	}

	ctx := context.Background()

	for _, testArg := range args {
		c := newTestingCache()
		rc, err := NewRouteCacheBuilder(c, 10).create()
		assert.NoError(t, err)

		wrappedProcessor := wrapProcessorFunc("/", testArg.processor, rc)

		response, err := wrappedProcessor(ctx, testArg.req)

		if testArg.err {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, response)
		}

	}

}

func TestHandlerWrapper(t *testing.T) {

	type arg struct {
		handler http.HandlerFunc
		req     *http.Request
		rsp     *ResponseReadWriter
	}

	args := []arg{
		{
			handler: func(writer http.ResponseWriter, request *http.Request) {
				i, err := writer.Write([]byte(request.RequestURI))
				assert.NoError(t, err)
				assert.True(t, i > 0)
			},
			rsp: NewResponseReadWriter(),
			req: &http.Request{RequestURI: "http://www.localhost.com"},
		},
	}

	for _, testArg := range args {
		c := newTestingCache()
		rc, err := NewRouteCacheBuilder(c, 10).create()
		assert.NoError(t, err)

		wrappedHandler := wrapHandlerFunc(testArg.handler, rc)

		wrappedHandler(testArg.rsp, testArg.req)

		assert.NoError(t, err)
		b, err := testArg.rsp.ReadAll()
		assert.NoError(t, err)
		assert.NotNil(t, b)
		assert.True(t, len(b) > 0)
	}

}

func TestRouteCacheImplementation_WithSingleRequest(t *testing.T) {

	cache := newTestingCache()

	var executions uint32

	routeBuilder := NewRouteBuilder("/path", func(context context.Context, request *Request) (response *Response, e error) {
		atomic.AddUint32(&executions, 1)
		return NewResponse("body"), nil
	}).WithRouteCachedBuilder(NewRouteCacheBuilder(cache, 10*time.Second)).MethodGet()

	ctx, cln := context.WithTimeout(context.Background(), 5*time.Second)

	port := 50023
	runRoute(ctx, t, routeBuilder, port)

	assertResponse(ctx, t, []http.Response{
		{
			Header: map[string][]string{cacheControlHeader: {"max-age=10"}},
			Body:   &bodyReader{body: "\"body\""},
		},
		{
			Header: map[string][]string{cacheControlHeader: {"max-age=10"}},
			Body:   &bodyReader{body: "\"body\""},
		},
	}, port)

	assertCacheState(t, *cache, cacheState{
		setOps: 1,
		getOps: 2,
		size:   1,
	})

	assert.Equal(t, executions, uint32(1))

	cln()

}

func TestRawRouteCacheImplementation_WithSingleRequest(t *testing.T) {

	cache := newTestingCache()

	var executions uint32

	routeBuilder := NewRawRouteBuilder("/path", func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddUint32(&executions, 1)
		i, err := writer.Write([]byte("\"body\""))
		assert.NoError(t, err)
		assert.True(t, i > 0)
	}).WithRouteCachedBuilder(NewRouteCacheBuilder(cache, 10*time.Second)).MethodGet()

	ctx, cln := context.WithTimeout(context.Background(), 5*time.Second)

	port := 50024
	runRoute(ctx, t, routeBuilder, port)

	assertResponse(ctx, t, []http.Response{
		{
			Header: map[string][]string{cacheControlHeader: {"max-age=10"}},
			Body:   &bodyReader{body: "\"body\""},
		},
		{
			Header: map[string][]string{cacheControlHeader: {"max-age=10"}},
			Body:   &bodyReader{body: "\"body\""},
		},
	}, port)

	assertCacheState(t, *cache, cacheState{
		setOps: 1,
		getOps: 2,
		size:   1,
	})

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
		cl, err := client.New()
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

	cl, err := client.New()
	assert.NoError(t, err)
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/path", port), nil)
	assert.NoError(t, err)

	for _, expectedResponse := range expected {
		response, err := cl.Do(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, expectedResponse.Header.Get(cacheControlHeader), response.Header.Get(cacheControlHeader))
		assert.True(t, response.Header.Get(eTagHeader) != "")
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
