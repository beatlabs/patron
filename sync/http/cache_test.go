package http

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/beatlabs/patron/sync"
)

func TestExtractCacheHeaders(t *testing.T) {

	type args struct {
		headers    map[string]string
		noCache    bool
		forceCache bool
		ttl        int64
	}

	// TODO : cover the extract headers functionality from 'real' http header samples

	params := []args{
		{
			headers:    map[string]string{CacheControlHeader: "max-age=10"},
			noCache:    false,
			forceCache: false,
			ttl:        -10,
		},
	}

	for _, param := range params {
		req := sync.NewRequest(map[string]string{}, nil, param.headers, nil)
		noCache, forceCache, ttl := extractCacheHeaders(req)
		assert.Equal(t, param.noCache, noCache)
		assert.Equal(t, param.forceCache, forceCache)
		assert.Equal(t, param.ttl, ttl)
	}

}

type testArgs struct {
	header       map[string]string
	fields       map[string]string
	response     *sync.Response
	timeInstance int64
	err          error
}

// TestCacheMaxAgeHeader tests the cache implementation
// for the same request,
// for header max-age
func TestCacheMaxAgeHeader(t *testing.T) {

	args := [][]testArgs{
		// cache expiration with max-age header
		{
			// initial request
			{
				fields:       map[string]string{"VALUE": "1"},
				header:       map[string]string{CacheControlHeader: "max-age=10"},
				response:     sync.NewResponse(10),
				timeInstance: 1,
				err:          nil,
			},
			// cache response
			{
				fields:       map[string]string{"VALUE": "1"},
				header:       map[string]string{CacheControlHeader: "max-age=10"},
				response:     sync.NewResponse(10),
				timeInstance: 9,
				err:          nil,
			},
			// still cached response because we are at the edge of the expiry e.g. 11 - 1 = 10
			{
				fields:       map[string]string{"VALUE": "1"},
				header:       map[string]string{CacheControlHeader: "max-age=10"},
				response:     sync.NewResponse(10),
				timeInstance: 11,
				err:          nil,
			},
			// new response because cache has expired
			{
				fields:       map[string]string{"VALUE": "1"},
				header:       map[string]string{CacheControlHeader: "max-age=10"},
				response:     sync.NewResponse(120),
				timeInstance: 12,
				err:          nil,
			},
			// make an extra request with the new cache value
			{
				fields:       map[string]string{"VALUE": "1"},
				header:       map[string]string{CacheControlHeader: "max-age=10"},
				response:     sync.NewResponse(120),
				timeInstance: 15,
				err:          nil,
			},
			// and another when the previous has expired 12 + 10 = 22
			{
				fields:       map[string]string{"VALUE": "1"},
				header:       map[string]string{CacheControlHeader: "max-age=10"},
				response:     sync.NewResponse(230),
				timeInstance: 23,
				err:          nil,
			},
		},
	}

	run(t, args)

}

// TestCacheMaxAgeHeader tests the cache implementation
// for the same request,
// for header max-age
func TestCacheMinFreshHeader(t *testing.T) {

	args := [][]testArgs{
		// cache expiration with max-age header
		{
			// initial request
			{
				fields:       map[string]string{"VALUE": "1"},
				header:       map[string]string{CacheControlHeader: "min-fresh=10"},
				response:     sync.NewResponse(10),
				timeInstance: 1,
				err:          nil,
			},
			// cache response
			{
				fields:       map[string]string{"VALUE": "1"},
				header:       map[string]string{CacheControlHeader: "max-age=10"},
				response:     sync.NewResponse(10),
				timeInstance: 9,
				err:          nil,
			},
			// still cached response because we are at the edge of the expiry e.g. 11 - 1 = 10
			{
				fields:       map[string]string{"VALUE": "1"},
				header:       map[string]string{CacheControlHeader: "max-age=10"},
				response:     sync.NewResponse(10),
				timeInstance: 11,
				err:          nil,
			},
			// new response because cache has expired
			{
				fields:       map[string]string{"VALUE": "1"},
				header:       map[string]string{CacheControlHeader: "max-age=10"},
				response:     sync.NewResponse(120),
				timeInstance: 12,
				err:          nil,
			},
			// make an extra request with the new cache value
			{
				fields:       map[string]string{"VALUE": "1"},
				header:       map[string]string{CacheControlHeader: "max-age=10"},
				response:     sync.NewResponse(120),
				timeInstance: 15,
				err:          nil,
			},
			// and another when the previous has expired 12 + 10 = 22
			{
				fields:       map[string]string{"VALUE": "1"},
				header:       map[string]string{CacheControlHeader: "max-age=10"},
				response:     sync.NewResponse(230),
				timeInstance: 23,
				err:          nil,
			},
		},
	}

	run(t, args)

}
func run(t *testing.T, args [][]testArgs) {
	handler := func(timeInstance int64) func(ctx context.Context, request *sync.Request) (*sync.Response, error) {
		return func(ctx context.Context, request *sync.Request) (*sync.Response, error) {
			i, err := strconv.Atoi(request.Fields["VALUE"])
			if err != nil {
				return nil, err
			}
			// return the specified parameter multiplied by the time instant
			return sync.NewResponse(i * 10 * int(timeInstance)), nil
		}
	}

	cache := &testingCache{cache: make(map[string]interface{})}

	for _, testArg := range args {
		for _, arg := range testArg {
			request := sync.NewRequest(arg.fields, nil, arg.header, nil)
			// initial request
			response, err := cacheHandler(handler(arg.timeInstance), cache, func() int64 {
				return arg.timeInstance
			})(context.Background(), request)
			if arg.err != nil {
				assert.Error(t, err)
				assert.NotNil(t, response)
				// TODO : assert type of error
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, arg.response, response)
			}
		}
	}
}

type testingCache struct {
	cache map[string]interface{}
}

func (t *testingCache) Get(key string) (interface{}, bool, error) {
	r, ok := t.cache[key]
	println(fmt.Sprintf("key = %v", key))
	println(fmt.Sprintf("r = %v", r))
	return r, ok, nil
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

func (t *testingCache) Set(key string, value interface{}) error {
	t.cache[key] = value
	println(fmt.Sprintf("key = %v", key))
	println(fmt.Sprintf("value = %v", value))
	return nil
}
