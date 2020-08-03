package cache

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
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
			headers: map[string]string{HeaderCacheControl: "max-age=10"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 1,
			},
			wrn: "",
		},
		// Header cannot be parsed
		{
			headers: map[string]string{HeaderCacheControl: "maxage=10"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
			},
			wrn: "",
		},
		// Header resets to minAge
		{
			headers: map[string]string{HeaderCacheControl: "max-age=twenty"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 1,
			},
			wrn: "max-age=5",
		},
		// Header resets to maxFresh e.g. maxAge - minAge
		{
			headers: map[string]string{HeaderCacheControl: "min-fresh=10"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 1,
			},
			wrn: "min-fresh=5",
		},
		// no Warning e.g. headers are within allowed values
		{
			headers: map[string]string{HeaderCacheControl: "min-fresh=5,max-age=5"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 2,
			},
			wrn: "",
		},
		// cache headers reset to min-age, note we still cache but send a Warning Header back
		{
			headers: map[string]string{HeaderCacheControl: "no-cache"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 1,
			},
			wrn: "max-age=5",
		},
		{
			headers: map[string]string{HeaderCacheControl: "no-store"},
			cfg: caheRequestCondition{
				noCache:    false,
				forceCache: false,
				validators: 1,
			},
			wrn: "max-age=5",
		},
	}

	for _, param := range params {
		header := param.headers[HeaderCacheControl]
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
	query        string
	timeInstance int64
}

// responseStruct emulates the patron http response,
// but this can be any struct in general
type responseStruct struct {
	Payload interface{}
	Header  map[string]string
}

func newRequestAt(timeInstant int64, ControlHeaders ...string) requestParams {
	params := requestParams{
		query:        "VALUE=1",
		timeInstance: timeInstant,
		header:       make(map[string]string),
	}
	if len(ControlHeaders) > 0 {
		params.header[HeaderCacheControl] = strings.Join(ControlHeaders, ",")
	}
	return params
}

func maxAgeHeader(value string) string {
	return fmt.Sprintf("%s=%s", headerCacheMaxAge, value)
}

func minFreshHeader(value string) string {
	return fmt.Sprintf("%s=%s", controlMinFresh, value)
}

type testArgs struct {
	routeConfig   routeConfig
	cache         cache.TTLCache
	requestParams requestParams
	response      *responseStruct
	metrics       testMetrics
	err           error
}

func testHeader(maxAge int64) map[string]string {
	header := make(map[string]string)
	header[HeaderCacheControl] = createCacheControlHeader(maxAge, 0)
	return header
}

func testHeaderWithWarning(maxAge int64, warning string) map[string]string {
	h := testHeader(maxAge)
	h[headerWarning] = warning
	return h
}

func TestMinAgeCache_WithoutClientHeader(t *testing.T) {

	rc := routeConfig{
		path: "/",
		age:  Age{Min: 1 * time.Second, Max: 10 * time.Second},
	}

	args := [][]testArgs{
		// cache expiration with max-age Header
		{
			// initial request, will fill up the cache
			{
				requestParams: newRequestAt(1),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 10, Header: testHeader(10)},
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
			// cache Response
			{
				requestParams: newRequestAt(9),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 10, Header: testHeader(2)},
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
			// still cached Response
			{
				requestParams: newRequestAt(11),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 10, Header: testHeader(0)},
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
			// new Response , due to expiry validator 10 + 1 - 12 < 0
			{
				requestParams: newRequestAt(12),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 120, Header: testHeader(10)},
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
		// this means , without client control headers we will always return a non-cached Response
		// without any proper age configuration
		age: Age{},
	}

	args := [][]testArgs{
		// cache expiration with max-age Header
		{
			// initial request, will fill up the cache
			{
				requestParams: newRequestAt(1),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 10},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {},
					},
				},
				err: nil,
			},
			// no cached Response
			{
				requestParams: newRequestAt(2),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 20},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {},
					},
				},
				err: nil,
			},
			// no cached Response
			{
				requestParams: newRequestAt(2, maxAgeHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 20},
				metrics: testMetrics{
					map[string]*metricState{
						"/": {},
					},
				},
				err: nil,
			},
			// no cached Response
			{
				requestParams: newRequestAt(2, minFreshHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 20},
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
		// cache expiration with max-age Header
		{
			// initial request, will fill up the cache
			{
				requestParams: newRequestAt(1, maxAgeHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 10, Header: testHeader(10)},
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
			// cached Response
			{
				requestParams: newRequestAt(3, maxAgeHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 10, Header: testHeader(8)},
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
			// new Response, because max-age > 9 - 1
			{
				requestParams: newRequestAt(9, maxAgeHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 90, Header: testHeader(10)},
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
			// cached Response right before the age threshold max-age == 14 - 9
			{
				requestParams: newRequestAt(14, maxAgeHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 90, Header: testHeader(5)},
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
			// new Response, because max-age > 15 - 9
			{
				requestParams: newRequestAt(15, maxAgeHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 150, Header: testHeader(10)},
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
		// cache expiration with max-age Header
		{
			// initial request, will fill up the cache
			{
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(30)},
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
			// cached Response
			{
				requestParams: newRequestAt(10, maxAgeHeader("10")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(20)},
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
			// cached Response
			{
				requestParams: newRequestAt(20, maxAgeHeader("20")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
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
			// new Response
			{
				requestParams: newRequestAt(20, maxAgeHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 200, Header: testHeader(30)},
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
			// cache Response
			{
				requestParams: newRequestAt(25, maxAgeHeader("25")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 200, Header: testHeader(25)},
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
		// cache expiration with max-age Header
		{
			// initial request, will fill up the cache
			{
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(5)},
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
				requestParams: newRequestAt(6, maxAgeHeader("100")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 60, Header: testHeader(5)},
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
		// cache expiration with max-age Header
		{
			// initial request, will fill up the cache
			{
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(30)},
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
				requestParams: newRequestAt(1, maxAgeHeader("0")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 10, Header: testHeader(30)},
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
		// cache expiration with max-age Header
		{
			// initial request, will fill up the cache
			{
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(30)},
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
			// cached Response still, because of minAge override
			// note : max-age=2 gets ignored
			{
				requestParams: newRequestAt(4, maxAgeHeader("2")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeaderWithWarning(26, "max-age=5")},
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
			// cached Response because of bigger max-age parameter
			{
				requestParams: newRequestAt(5, maxAgeHeader("20")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(25)},
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
			// new Response because of minAge floor
			{
				requestParams: newRequestAt(6, maxAgeHeader("3")),
				routeConfig:   rc,
				// note : no Warning because it s a new Response
				response: &responseStruct{Payload: 60, Header: testHeader(30)},
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
				requestParams: newRequestAt(0, minFreshHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
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
			// expecting cache Response, as value is still fresh : 5 - 0 == 5
			{
				requestParams: newRequestAt(5, minFreshHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(5)},
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
			// expecting new Response, as value is not fresh enough
			{
				requestParams: newRequestAt(6, minFreshHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 60, Header: testHeader(10)},
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
			// cache Response, as value is expired : 11 - 6 <= 5
			{
				requestParams: newRequestAt(11, minFreshHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 60, Header: testHeader(5)},
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
			// expecting new Response
			{
				requestParams: newRequestAt(12, minFreshHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 120, Header: testHeader(10)},
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
				requestParams: newRequestAt(0, minFreshHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
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
				requestParams: newRequestAt(1, minFreshHeader("100")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 10, Header: testHeader(10)},
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
				requestParams: newRequestAt(0, minFreshHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
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
			// expecting cache Response, as min-fresh is bounded by maxFresh configuration  parameter
			{
				requestParams: newRequestAt(5, minFreshHeader("100")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeaderWithWarning(5, "min-fresh=5")},
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
				requestParams: newRequestAt(0, maxAgeHeader("5"), minFreshHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
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
			// expecting cache Response, as value is still fresh : 5 - 0 == min-fresh and still young : 5 - 0 < max-age
			{
				requestParams: newRequestAt(5, maxAgeHeader("10"), minFreshHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(5)},
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
			// new Response, as value is not fresh enough : 6 - 0 > min-fresh
			{
				requestParams: newRequestAt(6, maxAgeHeader("10"), minFreshHeader("5")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 60, Header: testHeader(10)},
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
			// cached Response, as value is still fresh enough and still young
			{
				requestParams: newRequestAt(6, maxAgeHeader("8"), minFreshHeader("10")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 60, Header: testHeaderWithWarning(10, "min-fresh=5")},
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
			// new Response, as value is still fresh enough but too old
			{
				requestParams: newRequestAt(15, maxAgeHeader("8"), minFreshHeader("10")),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 150, Header: testHeader(10)},
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
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
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
				requestParams: newRequestAt(11),
				routeConfig: routeConfig{
					path: rc.path,
					hnd: func(now int64, key string) *response {
						return &response{
							Err: hndErr,
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
		hnd: func(now int64, key string) *response {
			return &response{
				Err: hndErr,
			}
		},
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: newRequestAt(0),
				routeConfig:   rc,
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
		instant: NowSeconds,
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
				cache:         cacheImpl,
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							errors:    1,
						},
					},
				},
			},
			// new Response, because of cache get error
			{
				requestParams: newRequestAt(1),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 10, Header: testHeader(10)},
				cache:         cacheImpl,
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
		instant: NowSeconds,
	}

	args := [][]testArgs{
		{
			// initial request
			{
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
				cache:         cacheImpl,
				metrics: testMetrics{
					map[string]*metricState{
						"/": {
							additions: 1,
							misses:    1,
						},
					},
				},
			},
			// new Response, because of cache get error
			{
				requestParams: newRequestAt(1),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 10, Header: testHeader(10)},
				cache:         cacheImpl,
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
					query:        "VALUE=1",
					timeInstance: 0,
					path:         "/1",
				},
				routeConfig: rc,
				response:    &responseStruct{Payload: 0, Header: testHeader(10)},
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
			// cached Response for the same path
			{
				requestParams: requestParams{
					query:        "VALUE=1",
					timeInstance: 1,
					path:         "/1",
				},
				routeConfig: rc,
				response:    &responseStruct{Payload: 0, Header: testHeader(9)},
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
					query:        "VALUE=1",
					timeInstance: 1,
					path:         "/2",
				},
				routeConfig: rc,
				response:    &responseStruct{Payload: 10, Header: testHeader(10)},
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
			// cached Response for second path
			{
				requestParams: requestParams{
					query:        "VALUE=1",
					timeInstance: 2,
					path:         "/2",
				},
				routeConfig: rc,
				response:    &responseStruct{Payload: 10, Header: testHeader(9)},
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
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
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
			// cached Response for same request parameter
			{
				requestParams: requestParams{
					query:        "VALUE=1",
					timeInstance: 1,
				},
				routeConfig: rc,
				response:    &responseStruct{Payload: 0, Header: testHeader(9)},
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
			// new Response for different request parameter
			{
				requestParams: requestParams{
					query:        "VALUE=2",
					timeInstance: 1,
				},
				routeConfig: rc,
				response:    &responseStruct{Payload: 20, Header: testHeader(10)},
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
			// cached Response for second request parameter
			{
				requestParams: requestParams{
					query:        "VALUE=2",
					timeInstance: 2,
				},
				routeConfig: rc,
				response:    &responseStruct{Payload: 20, Header: testHeader(9)},
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
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
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
			// expecting new Response, as we are using no-cache Header
			{
				requestParams: newRequestAt(5, "no-cache"),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 50, Header: testHeader(10)},
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
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
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
			// expecting cached Response, as we are using no-cache Header but are within the minAge limit
			{
				requestParams: newRequestAt(2, "no-cache"),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeaderWithWarning(8, "max-age=2")},
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
			// expecting new Response, as we are using no-cache Header
			{
				requestParams: newRequestAt(5, "no-cache"),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 50, Header: testHeader(10)},
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
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
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
			// expecting new Response, as we are using no-store Header
			{
				requestParams: newRequestAt(5, "no-store"),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 50, Header: testHeader(10)},
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
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
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
			// expecting cached Response, as we are using no-store Header but are within the minAge limit
			{
				requestParams: newRequestAt(2, "no-store"),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeaderWithWarning(8, "max-age=2")},
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
			// expecting new Response, as we are using no-store Header
			{
				requestParams: newRequestAt(5, "no-store"),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 50, Header: testHeader(10)},
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
				requestParams: newRequestAt(0),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(10)},
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
			// expecting cache Response, as min-fresh is bounded by maxFresh configuration  parameter
			{
				requestParams: newRequestAt(5, "only-if-cached"),
				routeConfig:   rc,
				response:      &responseStruct{Payload: 0, Header: testHeader(5)},
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

	monitor = &testMetrics{}

	// create a test request handler
	// that returns the current time instant times '10' multiplied by the VALUE parameter in the request
	exec := func(request requestParams) func(now int64, key string) *response {
		return func(now int64, key string) *response {
			i, err := strconv.Atoi(strings.Split(request.query, "=")[1])
			if err != nil {
				return &response{
					Err: err,
				}
			}
			response := &response{
				Response: handlerResponse{
					Bytes:  []byte(strconv.Itoa(i * 10 * int(request.timeInstance))),
					Header: make(map[string][]string),
				},
				Etag:      generateETag([]byte{}, int(now)),
				LastValid: request.timeInstance,
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

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?%s", path, arg.requestParams.query), nil)
			assert.NoError(t, err)
			propagateHeaders(arg.requestParams.header, req.Header)
			assert.NoError(t, err)
			request := toCacheHandlerRequest(req)

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

			NowSeconds = func() int64 {
				return arg.requestParams.timeInstance
			}

			routeCache, errs := NewRouteCache(ch, arg.routeConfig.age)
			assert.Empty(t, errs)

			response, err := handler(hnd, routeCache)(request)

			if arg.err != nil {
				assert.Error(t, err)
				assert.Nil(t, response)
				assert.Equal(t, err, arg.err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				payload, err := strconv.Atoi(string(response.Bytes))
				assert.NoError(t, err)
				assert.Equal(t, arg.response.Payload, payload)
				assertHeader(t, HeaderCacheControl, arg.response.Header, response.Header)
				assertHeader(t, headerWarning, arg.response.Header, response.Header)
				assert.NotNil(t, arg.response.Header[HeaderETagHeader])
				if !hasNoAgeConfig(int64(arg.routeConfig.age.Min), int64(arg.routeConfig.age.Max)) {
					assert.NotEmpty(t, response.Header[HeaderETagHeader])
				}
			}
			assertMetrics(t, arg.metrics, *monitor.(*testMetrics))
		}
	}
}

func propagateHeaders(header map[string]string, wHeader http.Header) {
	for k, h := range header {
		wHeader.Set(k, h)
	}
}

func assertHeader(t *testing.T, key string, expected map[string]string, actual http.Header) {
	if expected[key] == "" {
		assert.Empty(t, actual[key])
	} else {
		assert.Equal(t, expected[key], actual[key][0])
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
