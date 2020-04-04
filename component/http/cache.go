package http

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"net/http"
	"strconv"
	"strings"

	"github.com/beatlabs/patron/log"
)

// CacheHeader is an enum representing the header value
type CacheHeader int

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

	// cache control header values

	// maxStale specifies the staleness in seconds
	// that the client is willing to accept on a cached response
	maxStale = "max-stale"
	// minFresh specifies the minimum amount in seconds
	// that the cached object should have before it expires
	minFresh = "min-fresh"
	// noCache specifies that the client does not expect to get a cached response
	noCache = "no-cache"
	// noStore specifies that the client does not expect to get a cached response
	noStore = "no-store"
	// onlyIfCached specifies that the client wants a response ,
	// only if it is present in the cache
	onlyIfCached = "only-if-cached"
	// mustRevalidate signals to the client that the response might be stale
	mustRevalidate = "must-revalidate"
	// maxAge specifies the maximum age in seconds
	// - that the client is willing to accept cached objects for
	// (if it s part of the request headers)
	// - that the response object still has , before it expires in the cache
	// (if it s part of the response headers)
	maxAge = "max-age"
	// other response headers
	// eTagHeader specifies the hash of the cached object
	eTagHeader = "ETag"
	// warningHeader signals to the client that it's request ,
	// as defined by the headers , could not be served consistently.
	// The client must assume the the best-effort approach has been used to return any response
	// it can ignore the response or use it knowingly of the potential staleness involved
	warningHeader = "Warning"
)

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
func (r *cacheHandlerRequest) fromRequest(path string, req *Request) {
	r.path = path
	if req.Headers != nil {
		r.header = req.Headers[cacheControlHeader]
	}
	if req.Fields != nil {
		if query, err := json.Marshal(req.Fields); err == nil {
			r.query = string(query)
		}
	}
}

// fromRequest transforms the http Request object to the cache handler request
func (r *cacheHandlerRequest) fromHTTPRequest(req *http.Request) {
	if req.Header != nil {
		r.header = req.Header.Get(cacheControlHeader)
	}
	if req.URL != nil {
		r.path = req.URL.Path
		r.query = req.URL.RawQuery
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

// cacheHandler wraps the an execution logic with a cache layer
// exec is the processor func that the cache will wrap
// rc is the route cache implementation to be used
func cacheHandler(exec executor, rc *routeCache) func(request *cacheHandlerRequest) (response *cacheHandlerResponse, e error) {

	return func(request *cacheHandlerRequest) (response *cacheHandlerResponse, e error) {

		now := rc.instant()

		cfg := extractRequestHeaders(request.header, rc.minAge, rc.maxFresh)
		if cfg.expiryValidator == nil {
			cfg.expiryValidator = expiryCheck
		}
		key := extractRequestKey(request.path, request.query)

		rsp := getResponse(cfg, key, now, rc, exec)

		response = rsp.response
		e = rsp.err

		if e == nil {
			addResponseHeaders(now, response.header, rsp, rc.ttl)
			if !rsp.fromCache {
				saveResponse(key, rsp, rc)
			}
		}

		return
	}
}

// getResponse will get the appropriate response either using the cache or the executor,
// depending on the
func getResponse(cfg *cacheControl, key string, now int64, rc *routeCache, exec executor) *cachedResponse {
	var rsp *cachedResponse
	if cfg.noCache && !rc.staleResponse {
		rsp = exec(now, key)
	} else {
		rsp = getFromCache(key, rc)
		if rsp == nil {
			rc.metrics.miss(key)
			rsp = exec(now, key)
		} else if rsp.err != nil {
			rc.metrics.err(key)
			rsp = exec(now, key)
		} else if isValid, cx := isValid(now-rsp.lastValid, rc.ttl, append(cfg.validators, cfg.expiryValidator)...); !isValid {
			tmpRsp := exec(now, key)
			// if we could not retrieve a fresh response,
			// serve the last cached value, with a warning header
			if cfg.forceCache || (rc.staleResponse && tmpRsp.err != nil) {
				rsp.warning = "last-valid"
				rc.metrics.hit(key)
			} else {
				rsp = tmpRsp
				rc.metrics.evict(key, cx, now-rsp.lastValid)
			}
		} else {
			rsp.warning = cfg.warning
			rc.metrics.hit(key)
		}
	}
	return rsp
}

func isValid(age, ttl int64, validators ...validator) (bool, validationContext) {
	if len(validators) == 0 {
		return false, 0
	}
	for _, validator := range validators {
		if isValid, cx := validator(age, ttl); !isValid {
			return false, cx
		}
	}
	return true, 0
}

// getFromCache is the implementation that will provide a cachedResponse instance from the cache,
// if it exists
func getFromCache(key string, rc *routeCache) *cachedResponse {
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

// saveResponse caches the given response if required
func saveResponse(key string, rsp *cachedResponse, rc *routeCache) {
	if !rsp.fromCache && rsp.err == nil {
		if err := rc.cache.Set(key, rsp); err != nil {
			log.Errorf("could not cache response for request key %s %v", key, err)
		} else {
			rc.metrics.add(key)
		}
	}
}

// addResponseHeaders adds the appropriate headers according to the cachedResponse conditions
func addResponseHeaders(now int64, header map[string]string, rsp *cachedResponse, ttl int64) {
	header[eTagHeader] = rsp.etag
	header[cacheControlHeader] = createCacheControlHeader(ttl, now-rsp.lastValid)
	if rsp.warning != "" && rsp.fromCache {
		header[warningHeader] = rsp.warning
	} else {
		delete(header, warningHeader)
	}
}

// extractRequestKey generates a unique cache key based on the route path and the query parameters
func extractRequestKey(path, query string) string {
	return fmt.Sprintf("%s:%s", path, query)
}

// extractRequestHeaders extracts the client request headers allowing the client some control over the cache
func extractRequestHeaders(header string, minAge, maxFresh int64) *cacheControl {
	if header == "" {
		return &cacheControl{noCache: minAge == 0 && maxFresh == 0}
	}

	cfg := cacheControl{
		validators: make([]validator, 0),
	}

	wrn := make([]string, 0)

	for _, header := range strings.Split(header, ",") {
		keyValue := strings.Split(header, "=")
		headerKey := strings.ToLower(keyValue[0])
		switch headerKey {
		case maxStale:
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
			cfg.expiryValidator = func(age, ttl int64) (bool, validationContext) {
				return ttl-age+value >= 0, maxStaleValidation
			}
		case maxAge:
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
			cfg.validators = append(cfg.validators, func(age, ttl int64) (bool, validationContext) {
				return age <= value, maxAgeValidation
			})
		case minFresh:
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
			cfg.validators = append(cfg.validators, func(age, ttl int64) (bool, validationContext) {
				return ttl-age >= value, minFreshValidation
			})
		case noCache:
			/**
			return response if entity has changed
			e.g. (304 response if nothing has changed : 304 Not Modified)
			it SHOULD NOT include min-fresh, max-stale, or max-age.
			request should be accompanied by an ETag token
			*/
			fallthrough
		case noStore:
			/**
			no storage whatsoever
			*/
			if minAge == 0 && maxFresh == 0 {
				cfg.noCache = true
			} else {
				wrn = append(wrn, fmt.Sprintf("max-age=%d", minAge))
				cfg.validators = append(cfg.validators, func(age, ttl int64) (bool, validationContext) {
					return age <= minAge, maxAgeValidation
				})
			}
		case onlyIfCached:
			/**
			return only if is in cache , otherwise 504
			*/
			cfg.forceCache = true
		default:
			log.Warnf("unrecognised cache header %s", header)
		}
	}
	if len(wrn) > 0 {
		cfg.warning = strings.Join(wrn, ",")
	}

	return &cfg
}

func generateETag(key []byte, t int) string {
	return fmt.Sprintf("%d-%d", crc32.ChecksumIEEE(key), t)
}

func createCacheControlHeader(ttl, lastValid int64) string {
	mAge := ttl - lastValid
	if mAge < 0 {
		return mustRevalidate
	}
	return fmt.Sprintf("%s=%d", maxAge, ttl-lastValid)
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
