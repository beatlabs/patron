package http

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/beatlabs/patron/cache"

	"github.com/beatlabs/patron/log"
)

type validationContext int

const (
	// validation due to normal expiry in terms of ttl setting
	ttlValidation validationContext = iota + 1
	// maxAgeValidation represents a validation , happening due to max-age  header requirements
	maxAgeValidation
	// minFreshValidation represents a validation , happening due to min-fresh  header requirements
	minFreshValidation
	// maxStaleValidation represents a validation , happening due to max-stale header requirements
	maxStaleValidation

	// cacheControlHeader is the header key for cache related values
	// note : it is case-sensitive
	cacheControlHeader = "Cache-Control"

	cacheControlMaxStale     = "max-stale"
	cacheControlMinFresh     = "min-fresh"
	cacheControlNoCache      = "no-cache"
	cacheControlNoStore      = "no-store"
	cacheControlOnlyIfCached = "only-if-cached"
	cacheControlEmpty        = ""

	cacheHeaderMustRevalidate = "must-revalidate"
	cacheHeaderMaxAge         = "max-age"
	cacheHeaderETagHeader     = "ETag"
	cacheHeaderWarning        = "Warning"
)

var metrics cacheMetrics

func init() {
	metrics = newPrometheusMetrics()
}

// TimeInstant is a timing function
// returns the current time instant of the system's clock
// by default it can be `tine.Now().Unix()` ,
// but this abstraction allows also for non-linear implementations
type TimeInstant func() int64

// validator is a conditional function on an objects age and the configured ttl
type validator func(age, ttl int64) (bool, validationContext)

// expiryCheck is the main validator that checks that the entry has not expired e.g. is stale
var expiryCheck validator = func(age, ttl int64) (bool, validationContext) {
	return age <= ttl, ttlValidation
}

// cacheControl is the model of the request parameters regarding the cache control
type cacheControl struct {
	noCache         bool
	forceCache      bool
	warning         string
	validators      []validator
	expiryValidator validator
}

// cacheHandlerResponse is the dedicated response object for the cache handler
type cacheHandlerResponse struct {
	payload interface{}
	bytes   []byte
	header  map[string]string
}

// cacheHandlerRequest is the dedicated request object for the cache handler
type cacheHandlerRequest struct {
	header string
	path   string
	query  string
}

// fromRequest transforms the Request object to the cache handler request
func fromRequest(path string, req *Request) *cacheHandlerRequest {
	var header string
	if req.Headers != nil {
		header = req.Headers[cacheControlHeader]
	}
	var query string
	if req.Fields != nil {
		if fields, err := json.Marshal(req.Fields); err == nil {
			query = string(fields)
		}
	}
	return &cacheHandlerRequest{
		header: header,
		path:   path,
		query:  query,
	}
}

// fromHTTPRequest transforms the http Request object to the cache handler request
func fromHTTPRequest(req *http.Request) *cacheHandlerRequest {
	var header string
	if req.Header != nil {
		header = req.Header.Get(cacheControlHeader)
	}
	var path string
	var query string
	if req.URL != nil {
		path = req.URL.Path
		query = req.URL.RawQuery
	}
	return &cacheHandlerRequest{
		header: header,
		path:   path,
		query:  query,
	}
}

// cachedResponse is the struct representing an object retrieved or ready to be put into the route cache
type cachedResponse struct {
	response  *cacheHandlerResponse
	lastValid int64
	etag      string
	warning   string
	fromCache bool
	err       error
}

// executor is the function returning a cache response object from the underlying implementation
type executor func(now int64, key string) *cachedResponse

type cacheUpdate func(path, key string, rsp *cachedResponse)

// cacheHandler wraps the an execution logic with a cache layer
// exec is the processor func that the cache will wrap
// rc is the route cache implementation to be used
func cacheHandler(exec executor, rc *RouteCache) func(request *cacheHandlerRequest) (response *cacheHandlerResponse, e error) {

	saveToCache := determineCache(rc)

	return func(request *cacheHandlerRequest) (response *cacheHandlerResponse, e error) {

		now := rc.instant()

		key := extractRequestKey(request.path, request.query)

		var rsp *cachedResponse

		if hasNoAgeConfig(rc.age.min, rc.age.max) {
			rsp = exec(now, key)
			return rsp.response, rsp.err
		}

		cfg := extractRequestHeaders(request.header, rc.age.min, rc.age.max-rc.age.min)
		if cfg.expiryValidator == nil {
			cfg.expiryValidator = expiryCheck
		}

		rsp = getResponse(cfg, request.path, key, now, rc, exec)
		response = rsp.response
		e = rsp.err

		if e == nil {
			addResponseHeaders(now, response.header, rsp, rc.age.max)
			if !rsp.fromCache && !cfg.noCache {
				saveToCache(request.path, key, rsp)
			}
		}

		return
	}
}

// getResponse will get the appropriate response either using the cache or the executor,
// depending on the
func getResponse(cfg *cacheControl, path, key string, now int64, rc *RouteCache, exec executor) *cachedResponse {

	if cfg.noCache && !rc.staleResponse {
		return exec(now, key)
	}

	rsp := getFromCache(key, rc)
	if rsp == nil {
		metrics.miss(path)
		return exec(now, key)
	}
	if rsp.err != nil {
		metrics.err(path)
		return exec(now, key)
	}
	// if the object has expired
	if isValid, cx := isValid(now-rsp.lastValid, rc.age.max, append(cfg.validators, cfg.expiryValidator)...); !isValid {
		tmpRsp := exec(now, key)
		// if we could not retrieve a fresh response,
		// serve the last cached value, with a warning header
		if cfg.forceCache || (rc.staleResponse && tmpRsp.err != nil) {
			rsp.warning = "last-valid"
			metrics.hit(path)
		} else {
			rsp = tmpRsp
			metrics.evict(path, cx, now-rsp.lastValid)
		}
	} else {
		// add any warning generated while parsing the headers
		rsp.warning = cfg.warning
		metrics.hit(path)
	}

	return rsp
}

func isValid(age, maxAge int64, validators ...validator) (bool, validationContext) {
	if len(validators) == 0 {
		return false, 0
	}
	for _, validator := range validators {
		if isValid, cx := validator(age, maxAge); !isValid {
			return false, cx
		}
	}
	return true, 0
}

// getFromCache is the implementation that will provide a cachedResponse instance from the cache,
// if it exists
func getFromCache(key string, rc *RouteCache) *cachedResponse {
	if resp, ok, err := rc.cache.Get(key); ok && err == nil {
		if r, ok := resp.(*cachedResponse); ok {
			r.fromCache = true
			return r
		}
		return &cachedResponse{err: fmt.Errorf("could not parse cached response %v for key %s", resp, key)}
	} else if err != nil {
		return &cachedResponse{err: fmt.Errorf("could not read cache value for [ key = %v , err = %v ]", key, err)}
	}
	return nil
}

// determines if the cache can be used as a TTLCache
func determineCache(rc *RouteCache) cacheUpdate {
	if c, ok := rc.cache.(cache.TTLCache); ok {
		return func(path, key string, rsp *cachedResponse) {
			saveResponseWithTTL(path, key, rsp, c, time.Duration(rc.age.max)*time.Second)
		}
	}
	return func(path, key string, rsp *cachedResponse) {
		saveResponse(path, key, rsp, rc.cache)
	}
}

// saveResponse caches the given response if required
// the cachec implementation should handle the clean-up internally
func saveResponse(path, key string, rsp *cachedResponse, cache cache.Cache) {
	if !rsp.fromCache && rsp.err == nil {
		if err := cache.Set(key, rsp); err != nil {
			log.Errorf("could not cache response for request key %s %v", key, err)
		} else {
			metrics.add(path)
		}
	}
}

// saveResponseWithTTL caches the given response if required with a ttl
// as we are putting the objects in the cache, if its a TTL one, we need to manage the expiration on our own
func saveResponseWithTTL(path, key string, rsp *cachedResponse, cache cache.TTLCache, maxAge time.Duration) {
	if !rsp.fromCache && rsp.err == nil {
		if err := cache.SetTTL(key, rsp, maxAge); err != nil {
			log.Errorf("could not cache response for request key %s %v", key, err)
		} else {
			metrics.add(path)
		}
	}
}

// addResponseHeaders adds the appropriate headers according to the cachedResponse conditions
func addResponseHeaders(now int64, header map[string]string, rsp *cachedResponse, maxAge int64) {
	header[cacheHeaderETagHeader] = rsp.etag
	header[cacheControlHeader] = createCacheControlHeader(maxAge, now-rsp.lastValid)
	if rsp.warning != "" && rsp.fromCache {
		header[cacheHeaderWarning] = rsp.warning
	} else {
		delete(header, cacheHeaderWarning)
	}
}

// extractRequestKey generates a unique cache key based on the route path and the query parameters
func extractRequestKey(path, query string) string {
	return fmt.Sprintf("%s:%s", path, query)
}

// extractRequestHeaders extracts the client request headers allowing the client some control over the cache
func extractRequestHeaders(header string, minAge, maxFresh int64) *cacheControl {

	cfg := cacheControl{
		validators: make([]validator, 0),
	}

	wrn := make([]string, 0)

	for _, header := range strings.Split(header, ",") {
		keyValue := strings.Split(header, "=")
		headerKey := strings.ToLower(keyValue[0])
		switch headerKey {
		case cacheControlMaxStale:
			/**
			Indicates that the client is willing to accept a response that has
			exceeded its expiration time. If max-stale is assigned a value,
			then the client is willing to accept a response that has exceeded
			its expiration time by no more than the specified number of
			seconds. If no value is assigned to max-stale, then the client is
			willing to accept a stale response of any age.
			*/
			value, ok := parseValue(keyValue)
			if !ok || value < 0 {
				log.Debugf("invalid value for header '%s', defaulting to '0' ", keyValue)
				value = 0
			}
			cfg.expiryValidator = func(age, maxAge int64) (bool, validationContext) {
				return maxAge-age+value >= 0, maxStaleValidation
			}
		case cacheHeaderMaxAge:
			/**
			Indicates that the client is willing to accept a response whose
			age is no greater than the specified time in seconds. Unless max-
			stale directive is also included, the client is not willing to
			accept a stale response.
			*/
			value, ok := parseValue(keyValue)
			if !ok || value < 0 {
				log.Debugf("invalid value for header '%s', defaulting to '0' ", keyValue)
				value = 0
			}
			value, adjusted := min(value, minAge)
			if adjusted {
				wrn = append(wrn, fmt.Sprintf("max-age=%d", minAge))
			}
			cfg.validators = append(cfg.validators, func(age, maxAge int64) (bool, validationContext) {
				return age <= value, maxAgeValidation
			})
		case cacheControlMinFresh:
			/**
			Indicates that the client is willing to accept a response whose
			freshness lifetime is no less than its current age plus the
			specified time in seconds. That is, the client wants a response
			that will still be fresh for at least the specified number of
			seconds.
			*/
			value, ok := parseValue(keyValue)
			if !ok || value < 0 {
				log.Debugf("invalid value for header '%s', defaulting to '0' ", keyValue)
				value = 0
			}
			value, adjusted := max(value, maxFresh)
			if adjusted {
				wrn = append(wrn, fmt.Sprintf("min-fresh=%d", maxFresh))
			}
			cfg.validators = append(cfg.validators, func(age, maxAge int64) (bool, validationContext) {
				return maxAge-age >= value, minFreshValidation
			})
		case cacheControlNoCache:
			/**
			return response if entity has changed
			e.g. (304 response if nothing has changed : 304 Not Modified)
			it SHOULD NOT include min-fresh, max-stale, or max-age.
			request should be accompanied by an ETag token
			*/
			fallthrough
		case cacheControlNoStore:
			/**
			no storage whatsoever
			*/
			wrn = append(wrn, fmt.Sprintf("max-age=%d", minAge))
			cfg.validators = append(cfg.validators, func(age, maxAge int64) (bool, validationContext) {
				return age <= minAge, maxAgeValidation
			})
		case cacheControlOnlyIfCached:
			/**
			return only if is in cache , otherwise 504
			*/
			cfg.forceCache = true
		case cacheControlEmpty:
			// nothing to do here
		default:
			log.Warn("unrecognised cache header: '%s'", header)
		}
	}
	if len(wrn) > 0 {
		cfg.warning = strings.Join(wrn, ",")
	}

	return &cfg
}

func hasNoAgeConfig(minAge, maxFresh int64) bool {
	return minAge == 0 && maxFresh == 0
}

func generateETag(key []byte, t int) string {
	return fmt.Sprintf("%d-%d", crc32.ChecksumIEEE(key), t)
}

func createCacheControlHeader(ttl, lastValid int64) string {
	mAge := ttl - lastValid
	if mAge < 0 {
		return cacheHeaderMustRevalidate
	}
	return fmt.Sprintf("%s=%d", cacheHeaderMaxAge, ttl-lastValid)
}

func min(value, threshold int64) (int64, bool) {
	if value < threshold {
		return threshold, true
	}
	return value, false
}

func max(value, threshold int64) (int64, bool) {
	if threshold > 0 && value > threshold {
		return threshold, true
	}
	return value, false
}

func parseValue(keyValue []string) (int64, bool) {
	if len(keyValue) > 1 {
		value, err := strconv.ParseInt(keyValue[1], 10, 64)
		if err == nil {
			return value, true
		}
	}
	return 0, false
}
