package http

import (
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/beatlabs/patron/cache"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/log/zerolog"
)

func TestMain(m *testing.M) {

	err := log.Setup(zerolog.Create(log.DebugLevel), make(map[string]interface{}))

	if err != nil {
		os.Exit(1)
	}

	exitVal := m.Run()

	os.Exit(exitVal)

}

func TestExtractCacheHeaders(t *testing.T) {

	type caheRequestCondition struct {
		noCache         bool
		forceCache      bool
		validators      int
		expiryValidator bool
	}

	type args struct {
		cfg     caheRequestCondition
		headers map[string]string
		wrn     string
	}

	minAge := int64(5)
	maxAge := int64(10)

	params := []args{
		{
			headers: map[string]string{cacheControlHeader: "max-age=10"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 1,
			},
			wrn: "",
		},
		// header cannot be parsed
		{
			headers: map[string]string{cacheControlHeader: "maxage=10"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
			},
			wrn: "",
		},
		// header resets to minAge
		{
			headers: map[string]string{cacheControlHeader: "max-age=twenty"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 1,
			},
			wrn: "max-age=5",
		},
		// header resets to maxFresh e.g. maxAge - minAge
		{
			headers: map[string]string{cacheControlHeader: "min-fresh=10"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 1,
			},
			wrn: "min-fresh=5",
		},
		// no warning e.g. headers are within allowed values
		{
			headers: map[string]string{cacheControlHeader: "min-fresh=5,max-age=5"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 2,
			},
			wrn: "",
		},
		// cache headers reset to min-age, note we still cache but send a warning header back
		{
			headers: map[string]string{cacheControlHeader: "no-cache"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 1,
			},
			wrn: "max-age=5",
		},
		{
			headers: map[string]string{cacheControlHeader: "no-store"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 1,
			},
			wrn: "max-age=5",
		},
	}

	for _, param := range params {
		header := param.headers[cacheControlHeader]
		cfg := extractRequestHeaders(header, minAge, maxAge-minAge)
		assert.Equal(t, param.wrn, cfg.warning)
		assert.Equal(t, param.cfg.noCache, cfg.noCache)
		assert.Equal(t, param.cfg.forceCache, cfg.forceCache)
		assert.Equal(t, param.cfg.validators, len(cfg.validators))
		assert.Equal(t, param.cfg.expiryValidator, cfg.expiryValidator != nil)
	}

}

type routeConfig struct {
	path string
	hnd  executor
	age  Age
}

type requestParams struct {
	path         string
	header       map[string]string
	fields       map[string]string
	timeInstance int64
}

type testArgs struct {
	routeConfig   routeConfig
	cache         cache.TTLCache
	requestParams requestParams
	response      *Response
	metrics       testMetrics
	err           error
}

func testHeader(maxAge int64) map[string]string {
	header := make(map[string]string)
	header[cacheControlHeader] = createCacheControlHeader(maxAge, 0)
	return header
}

func testHeaderWithWarning(maxAge int64, warning string) map[string]string {
	h := testHeader(maxAge)
	h[cacheHeaderWarning] = warning
	return h
}

func TestMinAgeCache_WithoutClientHeader(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 1 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		// cache expiration with max-age header
		{
			// initial request, will fill up the cache
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 1,
				},
				routeConfig: rc,
				response:    &Response{Payload: 10, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// cache response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 9,
				},
				routeConfig: rc,
				response:    &Response{Payload: 10, Headers: testHeader(2)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
			// still cached response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 11,
				},
				routeConfig: rc,
				response:    &Response{Payload: 10, Headers: testHeader(0)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      2,
						},
					},
				},
				err: nil,
			},
			// new response , due to expiry validator 10 + 1 - 12 < 0
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 12,
				},
				routeConfig: rc,
				response:    &Response{Payload: 120, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    2,
							hits:      2,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

// No cache age configuration
// this effectively disables the cache
func TestNoAgeCache_WithoutClientHeader(t *testing.T) {

	rc := routeConfig{
		path: "/",
		// this means , without client control headers we will always return a non-cached response
		// without any proper age configuration
		age: Age{},
	}

	args := [][]testArgs{
		// cache expiration with max-age header
		{
			// initial request, will fill up the cache
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 1,
				},
				routeConfig: rc,
				response:    &Response{Payload: 10},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {},
					},
				},
				err: nil,
			},
			// no cached response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 2,
				},
				routeConfig: rc,
				response:    &Response{Payload: 20},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {},
					},
				},
				err: nil,
			},
			// no cached response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=5"},
					timeInstance: 2,
				},
				routeConfig: rc,
				response:    &Response{Payload: 20},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {},
					},
				},
				err: nil,
			},
			// no cached response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=5"},
					timeInstance: 2,
				},
				routeConfig: rc,
				response:    &Response{Payload: 20},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithConstantMaxAgeHeader(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 5 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		// cache expiration with max-age header
		{
			// initial request, will fill up the cache
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=5"},
					timeInstance: 1,
				},
				routeConfig: rc,
				response:    &Response{Payload: 10, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// cached response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=5"},
					timeInstance: 3,
				},
				routeConfig: rc,
				response:    &Response{Payload: 10, Headers: testHeader(8)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
			// new response, because max-age > 9 - 1
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=5"},
					timeInstance: 9,
				},
				routeConfig: rc,
				response:    &Response{Payload: 90, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							hits:      1,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
			// cached response right before the age threshold max-age == 14 - 9
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=5"},
					timeInstance: 14,
				},
				routeConfig: rc,
				response:    &Response{Payload: 90, Headers: testHeader(5)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							hits:      2,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
			// new response, because max-age > 15 - 9
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=5"},
					timeInstance: 15,
				},
				routeConfig: rc,
				response:    &Response{Payload: 150, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 3,
							misses:    1,
							hits:      2,
							evictions: 2,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithMaxAgeHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Max: 30 * time.Second},
	}

	args := [][]testArgs{
		// cache expiration with max-age header
		{
			// initial request, will fill up the cache
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(30)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// cached response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=10"},
					timeInstance: 10,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(20)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
			// cached response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=20"},
					timeInstance: 20,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      2,
						},
					},
				},
				err: nil,
			},
			// new response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=5"},
					timeInstance: 20,
				},
				routeConfig: rc,
				response:    &Response{Payload: 200, Headers: testHeader(30)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							hits:      2,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
			// cache response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=25"},
					timeInstance: 25,
				},
				routeConfig: rc,
				response:    &Response{Payload: 200, Headers: testHeader(25)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							hits:      3,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestMinAgeCache_WithHighMaxAgeHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Max: 5 * time.Second},
	}

	args := [][]testArgs{
		// cache expiration with max-age header
		{
			// initial request, will fill up the cache
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(5)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// despite the max-age request, the cache will refresh because of it's ttl
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=100"},
					timeInstance: 6,
				},
				routeConfig: rc,
				response:    &Response{Payload: 60, Headers: testHeader(5)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    2,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestNoMinAgeCache_WithLowMaxAgeHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Max: 30 * time.Second},
	}

	args := [][]testArgs{
		// cache expiration with max-age header
		{
			// initial request, will fill up the cache
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(30)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// a max-age=0 request will always refresh the cache,
			// if there is not minAge limit set
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=0"},
					timeInstance: 1,
				},
				routeConfig: rc,
				response:    &Response{Payload: 10, Headers: testHeader(30)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestMinAgeCache_WithMaxAgeHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 5 * time.Second, Max: 30 * time.Second},
	}

	args := [][]testArgs{
		// cache expiration with max-age header
		{
			// initial request, will fill up the cache
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(30)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// cached response still, because of minAge override
			// note : max-age=2 gets ignored
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=2"},
					timeInstance: 4,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeaderWithWarning(26, "max-age=5")},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
			// cached response because of bigger max-age parameter
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=20"},
					timeInstance: 5,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(25)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      2,
						},
					},
				},
				err: nil,
			},
			// new response because of minAge floor
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=3"},
					timeInstance: 6,
				},
				routeConfig: rc,
				// note : no warning because it s a new response
				response: &Response{Payload: 60, Headers: testHeader(30)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							hits:      2,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithConstantMinFreshHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 10 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=5"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// expecting cache response, as value is still fresh : 5 - 0 == 5
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=5"},
					timeInstance: 5,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(5)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
			// expecting new response, as value is not fresh enough
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=5"},
					timeInstance: 6,
				},
				routeConfig: rc,
				response:    &Response{Payload: 60, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							hits:      1,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
			// cache response, as value is expired : 11 - 6 <= 5
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=5"},
					timeInstance: 11,
				},
				routeConfig: rc,
				response:    &Response{Payload: 60, Headers: testHeader(5)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							hits:      2,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
			// expecting new response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=5"},
					timeInstance: 12,
				},
				routeConfig: rc,
				response:    &Response{Payload: 120, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 3,
							misses:    1,
							hits:      2,
							evictions: 2,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestNoMaxFreshCache_WithLargeMinFreshHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 10 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=5"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=100"},
					timeInstance: 1,
				},
				routeConfig: rc,
				response:    &Response{Payload: 10, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestMaxAgeCache_WithMinFreshHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		// Note  this is a bad config
		age: Age{Min: 5 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=5"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// expecting cache response, as min-fresh is bounded by maxFresh configuration  parameter
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=100"},
					timeInstance: 5,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeaderWithWarning(5, "min-fresh=5")},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithMixedHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 5 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=5,max-age=5"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// expecting cache response, as value is still fresh : 5 - 0 == min-fresh and still young : 5 - 0 < max-age
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=5,max-age=10"},
					timeInstance: 5,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(5)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
			// new response, as value is not fresh enough : 6 - 0 > min-fresh
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=5,max-age=10"},
					timeInstance: 6,
				},
				routeConfig: rc,
				response:    &Response{Payload: 60, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							hits:      1,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
			// cached response, as value is still fresh enough and still young
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=10,max-age=8"},
					timeInstance: 6,
				},
				routeConfig: rc,
				response:    &Response{Payload: 60, Headers: testHeaderWithWarning(10, "min-fresh=5")},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							hits:      2,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
			// new response, as value is still fresh enough but too old
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "min-fresh=10,max-age=8"},
					timeInstance: 15,
				},
				routeConfig: rc,
				response:    &Response{Payload: 150, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 3,
							misses:    1,
							hits:      2,
							evictions: 2,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithHandlerErrorWithoutHeaders(t *testing.T) {

	hndErr := errors.New("error encountered on handler")

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 10 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
			},
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 11,
				},
				routeConfig: routeConfig{
					path: rc.path,
					hnd: func(now int64, key string) *cachedResponse {
						return &cachedResponse{
							err: hndErr,
						}
					},
					age: rc.age,
				},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    2,
						},
					},
				},
				err: hndErr,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithHandlerErr(t *testing.T) {

	hndErr := errors.New("error encountered on handler")

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 10 * time.Second, Max: 10 * time.Second},
		hnd: func(now int64, key string) *cachedResponse {
			return &cachedResponse{
				err: hndErr,
			}
		},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							misses: 1,
						},
					},
				},
				err: hndErr,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithCacheGetErr(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 10 * time.Second, Max: 10 * time.Second},
	}

	cacheImpl := &testingCache{
		cache:   make(map[string]testingCacheEntity),
		getErr:  errors.New("get error"),
		instant: now,
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				cache:       cacheImpl,
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							errors:    1,
						},
					},
				},
			},
			// new response, because of cache get error
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 1,
				},
				routeConfig: rc,
				response:    &Response{Payload: 10, Headers: testHeader(10)},
				cache:       cacheImpl,
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							errors:    2,
						},
					},
				},
			},
		}}
	assertCache(t, args)

	assert.Equal(t, 2, cacheImpl.getCount)
	assert.Equal(t, 2, cacheImpl.setCount)
}

func TestCache_WithCacheSetErr(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 10 * time.Second, Max: 10 * time.Second},
	}

	cacheImpl := &testingCache{
		cache:   make(map[string]testingCacheEntity),
		setErr:  errors.New("set error"),
		instant: now,
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				cache:       cacheImpl,
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
			},
			// new response, because of cache get error
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 1,
				},
				routeConfig: rc,
				response:    &Response{Payload: 10, Headers: testHeader(10)},
				cache:       cacheImpl,
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    2,
						},
					},
				},
			},
		},
	}
	assertCache(t, args)

	assert.Equal(t, 2, cacheImpl.getCount)
	assert.Equal(t, 2, cacheImpl.setCount)
}

func TestCache_WithMixedPaths(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 10 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
					path:         "/1",
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/1": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// cached response for the same path
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 1,
					path:         "/1",
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(9)},
				metrics: testMetrics{
					map[string]*metricState{
						"/1": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
			// initial request for second path
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 1,
					path:         "/2",
				},
				routeConfig: rc,
				response:    &Response{Payload: 10, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/1": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
						"/2": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// cached response for second path
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 2,
					path:         "/2",
				},
				routeConfig: rc,
				response:    &Response{Payload: 10, Headers: testHeader(9)},
				metrics: testMetrics{
					map[string]*metricState{
						"/1": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
						"/2": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithMixedRequestParameters(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 10 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// cached response for same request parameter
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 1,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(9)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
			// new response for different request parameter
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "2"},
					timeInstance: 1,
				},
				routeConfig: rc,
				response:    &Response{Payload: 20, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    2,
							hits:      1,
						},
					},
				},
				err: nil,
			},
			// cached response for second request parameter
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "2"},
					timeInstance: 2,
				},
				routeConfig: rc,
				response:    &Response{Payload: 20, Headers: testHeader(9)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    2,
							hits:      2,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestZeroAgeCache_WithNoCacheHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Max: 10 * time.Second},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// expecting new response, as we are using no-cache header
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "no-cache"},
					timeInstance: 5,
				},
				routeConfig: rc,
				response:    &Response{Payload: 50, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestMinAgeCache_WithNoCacheHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 2 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// expecting cached response, as we are using no-cache header but are within the minAge limit
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "no-cache"},
					timeInstance: 2,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeaderWithWarning(8, "max-age=2")},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
			// expecting new response, as we are using no-cache header
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "no-cache"},
					timeInstance: 5,
				},
				routeConfig: rc,
				response:    &Response{Payload: 50, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							hits:      1,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestZeroAgeCache_WithNoStoreHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Max: 10 * time.Second},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// expecting new response, as we are using no-store header
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "no-store"},
					timeInstance: 5,
				},
				routeConfig: rc,
				response:    &Response{Payload: 50, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestMinAgeCache_WithNoStoreHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 2 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// expecting cached response, as we are using no-store header but are within the minAge limit
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "no-store"},
					timeInstance: 2,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeaderWithWarning(8, "max-age=2")},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
			// expecting new response, as we are using no-store header
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "no-store"},
					timeInstance: 5,
				},
				routeConfig: rc,
				response:    &Response{Payload: 50, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 2,
							misses:    1,
							hits:      1,
							evictions: 1,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithForceCacheHeaders(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 10 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
				err: nil,
			},
			// expecting cache response, as min-fresh is bounded by maxFresh configuration  parameter
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "only-if-cached"},
					timeInstance: 5,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(5)},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
							hits:      1,
						},
					},
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func assertCache(t *testing.T, args [][]testArgs) {

	metrics = &testMetrics{}

	// create a test request handler
	// that returns the current time instant times '10' multiplied by the VALUE parameter in the request
	exec := func(request requestParams) func(now int64, key string) *cachedResponse {
		return func(now int64, key string) *cachedResponse {
			i, err := strconv.Atoi(request.fields["VALUE"])
			if err != nil {
				return &cachedResponse{
					err: err,
				}
			}
			response := &cachedResponse{
				response: &cacheHandlerResponse{
					payload: i * 10 * int(request.timeInstance),
					header:  make(map[string]string),
				},
				etag:      generateETag([]byte{}, int(now)),
				lastValid: request.timeInstance,
			}
			return response
		}
	}

	// test cache implementation
	cacheIml := newTestingCache()

	for _, testArg := range args {
		for _, arg := range testArg {

			path := arg.routeConfig.path
			if arg.requestParams.path != "" {
				path = arg.requestParams.path
			}

			request := fromRequest(path, NewRequest(arg.requestParams.fields, nil, arg.requestParams.header, nil))

			var hnd executor
			if arg.routeConfig.hnd != nil {
				hnd = arg.routeConfig.hnd
			} else {
				hnd = exec(arg.requestParams)
			}

			var ch cache.TTLCache
			if arg.cache != nil {
				ch = arg.cache
			} else {
				ch = cacheIml
				cacheIml.instant = func() int64 {
					return arg.requestParams.timeInstance
				}
			}

			now = func() int64 {
				return arg.requestParams.timeInstance
			}

			response, err := cacheHandler(hnd, &routeCache{
				cache: ch,
				age:   arg.routeConfig.age.toAgeInSeconds(),
			})(request)

			if arg.err != nil {
				assert.Error(t, err)
				assert.Nil(t, response)
				assert.Equal(t, err, arg.err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, arg.response.Payload, response.payload)
				assert.Equal(t, arg.response.Headers[cacheControlHeader], response.header[cacheControlHeader])
				assert.Equal(t, arg.response.Headers[cacheHeaderWarning], response.header[cacheHeaderWarning])
				assert.NotNil(t, arg.response.Headers[cacheHeaderETagHeader])
				if !hasNoAgeConfig(int64(arg.routeConfig.age.Min), int64(arg.routeConfig.age.Max)) {
					assert.False(t, response.header[cacheHeaderETagHeader] == "")
				}
			}
			assertMetrics(t, arg.metrics, *metrics.(*testMetrics))
		}
	}
}

func assertMetrics(t *testing.T, expected, actual testMetrics) {
	for k, v := range expected.values {
		if actual.values == nil {
			assert.Equal(t, v, &metricState{})
		} else {
			assert.Equal(t, v, actual.values[k])
		}
	}
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
	instant  timeInstant
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

type testMetrics struct {
	values map[string]*metricState
}

type metricState struct {
	additions int
	misses    int
	evictions int
	hits      int
	errors    int
}

func (m *testMetrics) init(path string) {
	if m.values == nil {
		m.values = make(map[string]*metricState)
	}
	if _, exists := m.values[path]; !exists {

		m.values[path] = &metricState{}
	}
}

func (m *testMetrics) add(path string) {
	m.init(path)
	m.values[path].additions++
}

func (m *testMetrics) miss(path string) {
	m.init(path)
	m.values[path].misses++
}

func (m *testMetrics) hit(path string) {
	m.init(path)
	m.values[path].hits++
}

func (m *testMetrics) err(path string) {
	m.init(path)
	m.values[path].errors++
}

func (m *testMetrics) evict(path string, context validationContext, age int64) {
	m.init(path)
	m.values[path].evictions++
}
