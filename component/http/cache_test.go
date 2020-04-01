package http

import (
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/beatlabs/patron/cache"

	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/log/zerolog"

	"github.com/stretchr/testify/assert"
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

	// TODO : cover the extract headers functionality from 'real' http header samples

	minAge := int64(0)
	minFresh := int64(0)

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
			wrn: "",
		},
		{
			headers: map[string]string{cacheControlHeader: "min-fresh=10"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 1,
			},
			wrn: "",
		},
		{
			headers: map[string]string{cacheControlHeader: "min-fresh=5,max-age=5"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 2,
			},
			wrn: "",
		},
		{
			headers: map[string]string{cacheControlHeader: "max-stale=5"},
			cfg: caheRequestCondition{
				noCache:         false,
				forceCache:      false,
				expiryValidator: true,
			},
			wrn: "",
		},
		{
			headers: map[string]string{cacheControlHeader: "no-cache"},
			cfg: caheRequestCondition{
				noCache:    true,
				forceCache: false,
			},
			wrn: "",
		},
		{
			headers: map[string]string{cacheControlHeader: "no-store"},
			cfg: caheRequestCondition{
				noCache:    true,
				forceCache: false,
			},
			wrn: "",
		},
	}

	for _, param := range params {
		header := param.headers[cacheControlHeader]
		cfg, wrn := extractCacheHeaders(header, minAge, minFresh)
		assert.Equal(t, param.wrn, wrn)
		assert.Equal(t, param.cfg.noCache, cfg.noCache)
		assert.Equal(t, param.cfg.forceCache, cfg.forceCache)
		assert.Equal(t, param.cfg.validators, len(cfg.validators))
		assert.Equal(t, param.cfg.expiryValidator, cfg.expiryValidator != nil)
	}

}

type routeConfig struct {
	path          string
	ttl           int64
	hnd           executor
	minAge        int64
	maxFresh      int64
	staleResponse bool
}

type requestParams struct {
	path         string
	header       map[string]string
	fields       map[string]string
	timeInstance int64
}

type metricState struct {
	additions int
	misses    int
	evictions int
	hits      int
}

func (m *metricState) add(n metricState) {
	m.evictions += n.evictions
	m.additions += n.additions
	m.misses += n.misses
	m.hits += n.hits
}

type testArgs struct {
	routeConfig   routeConfig
	cache         cache.Cache
	requestParams requestParams
	response      *Response
	metrics       metricState
	err           error
}

func testHeader(maxAge int64) map[string]string {
	header := make(map[string]string)
	header[cacheControlHeader] = createCacheControlHeader(maxAge, 0)
	return header
}

func testHeaderWithWarning(maxAge int64, warning string) map[string]string {
	h := testHeader(maxAge)
	h[warningHeader] = warning
	return h
}

func TestMinAgeCache_WithoutClientHeader(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        1, // to avoid no-cache
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					additions: 1,
					evictions: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestNoMinAgeCache_WithoutClientHeader(t *testing.T) {

	rc := routeConfig{
		path:   "/",
		ttl:    10,
		minAge: 0, // min age is set to '0',
		// this means , without client control headers we will always return a non-cached response
		// despite the ttl parameter
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
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
				response:    &Response{Payload: 20, Headers: testHeader(10)},
				metrics: metricState{
					additions: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestNoMinAgeCache_WithMaxAgeHeader(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        0,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
				},
				err: nil,
			},
			// cached response, because of the max-age header
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=10"},
					timeInstance: 3,
				},
				routeConfig: rc,
				response:    &Response{Payload: 10, Headers: testHeader(8)},
				metrics: metricState{
					hits: 1,
				},
				err: nil,
			},
			// new response, because of missing header, and minAge == 0
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 9,
				},
				routeConfig: rc,
				response:    &Response{Payload: 90, Headers: testHeader(10)},
				metrics: metricState{
					additions: 1,
				},
				err: nil,
			},
			// new cached response , because max-age header again
			// note : because of the cache refresh triggered by the previous call we see the last cached value
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-age=10"},
					timeInstance: 14,
				},
				routeConfig: rc,
				response:    &Response{Payload: 90, Headers: testHeader(5)},
				metrics: metricState{
					hits: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithConstantMaxAgeHeader(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        5,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					additions: 1,
					evictions: 1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					additions: 1,
					evictions: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithMaxAgeHeaders(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           30,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					additions: 1,
					evictions: 1,
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
				metrics: metricState{
					hits: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestMinAgeCache_WithHighMaxAgeHeaders(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           5,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
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
				metrics: metricState{
					additions: 1,
					evictions: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestNoMinAgeCache_WithLowMaxAgeHeaders(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           30,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
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
				metrics: metricState{
					additions: 1,
					evictions: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestMinAgeCache_WithMaxAgeHeaders(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           30,
		minAge:        5,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					additions: 1,
					evictions: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithConstantMinFreshHeaders(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        10,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					additions: 1,
					evictions: 1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					additions: 1,
					evictions: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestNoMaxFreshCache_WithExtremeMinFreshHeaders(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        10,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					additions: 1,
					evictions: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestMaxFreshCache_WithMinFreshHeaders(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        10,
		maxFresh:      5,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					hits: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithConstantMaxStaleHeader(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        10,
		staleResponse: false,
	}

	args := [][]testArgs{
		{
			// initial request, will fill up the cache
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 0,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(10)},
				metrics: metricState{
					additions: 1,
					misses:    1,
				},
				err: nil,
			},
			// cached response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-stale=5"},
					timeInstance: 3,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(7)},
				metrics: metricState{
					hits: 1,
				},
				err: nil,
			},
			// cached response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-stale=5"},
					timeInstance: 8,
				},
				routeConfig: rc,
				response:    &Response{Payload: 0, Headers: testHeader(2)},
				metrics: metricState{
					hits: 1,
				},
				err: nil,
			},
			// cached response , still stale threshold not breached , 12 - 0 <= 10 + 5
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-stale=5"},
					timeInstance: 15,
				},
				routeConfig: rc,
				// note : we are also getting a must-revalidate header
				response: &Response{Payload: 0, Headers: testHeader(-5)},
				metrics: metricState{
					hits: 1,
				},
				err: nil,
			},
			// new response
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					header:       map[string]string{cacheControlHeader: "max-stale=5"},
					timeInstance: 16,
				},
				routeConfig: rc,
				response:    &Response{Payload: 160, Headers: testHeader(10)},
				metrics: metricState{
					additions: 1,
					evictions: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithMixedHeaders(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        5,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					additions: 1,
					evictions: 1,
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
				response:    &Response{Payload: 60, Headers: testHeader(10)},
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					additions: 1,
					evictions: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestStaleCache_WithHandlerErrorWithoutHeaders(t *testing.T) {

	hndErr := errors.New("error encountered on handler")

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        10,
		maxFresh:      10,
		staleResponse: true,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
				},
			},
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 11,
				},
				routeConfig: routeConfig{
					path: rc.path,
					ttl:  rc.ttl,
					hnd: func(now int64, key string) *cachedResponse {
						return &cachedResponse{
							err: hndErr,
						}
					},
					minAge:        rc.minAge,
					maxFresh:      rc.maxFresh,
					staleResponse: rc.staleResponse,
				},
				metrics: metricState{
					hits: 1,
				},
				response: &Response{Payload: 0, Headers: testHeaderWithWarning(-1, "last-valid")},
			},
		},
	}
	assertCache(t, args)
}

func TestNoStaleCache_WithHandlerErrorWithoutHeaders(t *testing.T) {

	hndErr := errors.New("error encountered on handler")

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        10,
		maxFresh:      10,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
				},
			},
			{
				requestParams: requestParams{
					fields:       map[string]string{"VALUE": "1"},
					timeInstance: 11,
				},
				routeConfig: routeConfig{
					path: rc.path,
					ttl:  rc.ttl,
					hnd: func(now int64, key string) *cachedResponse {
						return &cachedResponse{
							err: hndErr,
						}
					},
					minAge:        rc.minAge,
					maxFresh:      rc.maxFresh,
					staleResponse: rc.staleResponse,
				},
				metrics: metricState{
					evictions: 1,
				},
				err: hndErr,
			},
		},
	}
	assertCache(t, args)
}

// TODO : test stale response for error (with Warning)

func TestCache_WithHandlerErr(t *testing.T) {

	hndErr := errors.New("error encountered on handler")

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        10,
		maxFresh:      10,
		staleResponse: false,
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
				metrics: metricState{
					misses: 1,
				},
				err: hndErr,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithCacheGetErr(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        10,
		maxFresh:      10,
		staleResponse: false,
	}

	cacheImpl := &testingCache{
		cache:  make(map[string]interface{}),
		getErr: errors.New("get error"),
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
				},
			},
		},
	}
	assertCache(t, args)

	assert.Equal(t, 2, cacheImpl.getCount)
	assert.Equal(t, 2, cacheImpl.setCount)

}

func TestCache_WithCacheSetErr(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        10,
		maxFresh:      10,
		staleResponse: false,
	}

	cacheImpl := &testingCache{
		cache:  make(map[string]interface{}),
		setErr: errors.New("set error"),
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
		path:          "/",
		ttl:           10,
		minAge:        10,
		maxFresh:      10,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					hits: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

func TestCache_WithMixedRequestParameters(t *testing.T) {

	rc := routeConfig{
		path:          "/",
		ttl:           10,
		minAge:        10,
		maxFresh:      10,
		staleResponse: false,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					hits: 1,
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
				metrics: metricState{
					additions: 1,
					misses:    1,
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
				metrics: metricState{
					hits: 1,
				},
				err: nil,
			},
		},
	}
	assertCache(t, args)
}

// TODO  : test no-cache
// TODO : test no-store
// TODO : test only-if-cached

func assertCache(t *testing.T, args [][]testArgs) {

	chMetrics := NewPrometheusMetrics("path", 0)
	chMetrics.reset()

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
			return &cachedResponse{
				response: &cacheHandlerResponse{
					payload: i * 10 * int(request.timeInstance),
					header:  make(map[string]string),
				},
				etag:      generateETag([]byte{}, int(now)),
				lastValid: request.timeInstance,
			}
		}
	}

	// test cache implementation
	cacheIml := newTestingCache()

	argMetrics := metricState{}

	for _, testArg := range args {
		for _, arg := range testArg {

			path := arg.routeConfig.path
			if arg.requestParams.path != "" {
				path = arg.requestParams.path
			}

			request := &cacheHandlerRequest{}
			request.fromRequest(path, NewRequest(arg.requestParams.fields, nil, arg.requestParams.header, nil))

			var hnd executor
			if arg.routeConfig.hnd != nil {
				hnd = arg.routeConfig.hnd
			} else {
				hnd = exec(arg.requestParams)
			}

			var ch cache.Cache
			if arg.cache != nil {
				ch = arg.cache
			} else {
				ch = cacheIml
			}

			response, err := cacheHandler(hnd, &routeCache{
				cache: ch,
				instant: func() int64 {
					return arg.requestParams.timeInstance
				},
				ttl:           arg.routeConfig.ttl,
				minAge:        arg.routeConfig.minAge,
				maxFresh:      arg.routeConfig.maxFresh,
				staleResponse: arg.routeConfig.staleResponse,
				metrics:       chMetrics,
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
				assert.Equal(t, arg.response.Headers[warningHeader], response.header[warningHeader])
				assert.NotNil(t, arg.response.Headers[eTagHeader])
				assert.False(t, response.header[eTagHeader] == "")
			}
			// we provide the diff
			argMetrics.add(arg.metrics)
			// we assert the sum
			assertPrometheusMetrics(t, argMetrics, chMetrics)
		}
	}
}

func assertPrometheusMetrics(t *testing.T, mState metricState, metrics *PrometheusMetrics) {

	assertMetric(t, mState.misses, metrics.misses)
	assertMetric(t, mState.additions, metrics.additions)
	assertMetric(t, mState.hits, metrics.hits)
	assertMetric(t, mState.evictions, metrics.evictions)

}

func assertMetric(t *testing.T, value int, c prometheus.Collector) {
	if value > 0 {
		v := testutil.ToFloat64(c)
		assert.Equal(t, float64(value), v)
	} else {
		assertPanic(t, func() {
			testutil.ToFloat64(c)
		})
	}
}

func assertPanic(t *testing.T, exec func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	exec()
}

type testingCache struct {
	cache    map[string]interface{}
	getCount int
	setCount int
	getErr   error
	setErr   error
}

func newTestingCache() *testingCache {
	return &testingCache{cache: make(map[string]interface{})}
}

func (t *testingCache) Get(key string) (interface{}, bool, error) {
	t.getCount++
	if t.getErr != nil {
		return nil, false, t.getErr
	}
	r, ok := t.cache[key]
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
	t.setCount++
	if t.setErr != nil {
		return t.getErr
	}
	t.cache[key] = value
	return nil
}

func (t *testingCache) size() int {
	return len(t.cache)
}
