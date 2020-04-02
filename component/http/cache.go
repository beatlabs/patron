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

// TODO : wrap up implementation

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

// executor is the function returning a cache response object from the underlying implementation
type executor func(now int64, key string) *cachedResponse

// cacheHandler wraps the an execution logic with a cache layer
// exec is the processor func that the cache will wrap
// rc is the route cache implementation to be used
func cacheHandler(exec executor, rc *routeCache) func(request *cacheHandlerRequest) (response *cacheHandlerResponse, e error) {

	return func(request *cacheHandlerRequest) (response *cacheHandlerResponse, e error) {

		now := rc.instant()

		cfg, warning := extractCacheHeaders(request.header, rc.minAge, rc.maxFresh)
		if cfg.expiryValidator == nil {
			cfg.expiryValidator = expiryCheck
		}
		key := extractRequestKey(request.path, request.query)
		var rsp *cachedResponse
		var fromCache bool

		// explore the cache
		if cfg.noCache && !rc.staleResponse {
			// need to execute the handler always
			rsp = exec(now, key)
		} else {
			// lets check the cache if we have anything for the given key
			if rsp = cacheRetriever(key, rc, now); rsp == nil {
				rc.metrics.miss(key)
				// we have not encountered this key before
				rsp = exec(now, key)
			} else {
				expiry := rc.ttl
				age := now - rsp.lastValid
				if isValid, cx := isValid(age, expiry, append(cfg.validators, cfg.expiryValidator)...); !isValid {
					tmpRsp := exec(now, key)
					// if we could not generate a fresh response, serve the last cached value,
					// with a warning header
					if rc.staleResponse && tmpRsp.err != nil {
						warning = "last-valid"
						fromCache = true
						rc.metrics.hit(key)
					} else {
						rsp = tmpRsp
						rc.metrics.evict(key, cx, age)
					}
				} else {
					fromCache = true
					rc.metrics.hit(key)
				}
			}
		}
		// TODO : use the forceCache parameter
		if cfg.forceCache {
			// return empty response if we have rc-only responseHeaders present
			return &cacheHandlerResponse{payload: []byte{}}, nil
		}

		response = rsp.response
		println(fmt.Sprintf("rsp = %v", rsp))
		e = rsp.err

		// we cache response only if we did not retrieve it from the cache itself
		// and if there was no error
		if !fromCache && e == nil {
			if err := rc.cache.Set(key, rsp); err != nil {
				log.Errorf("could not cache response for request key %s %v", key, err)
			}
			rc.metrics.add(key)
		}

		// TODO : abstract into method
		if e == nil {
			response.header[eTagHeader] = rsp.etag
			response.header[cacheControlHeader] = createCacheControlHeader(rc.ttl, now-rsp.lastValid)
			if warning != "" && fromCache {
				response.header[warningHeader] = warning
			} else {
				delete(response.header, warningHeader)
			}
		}

		return
	}
}

// cacheRetriever is the implementation that will provide a cachedResponse instance from the cache,
// if it exists
var cacheRetriever = func(key string, rc *routeCache, now int64) *cachedResponse {
	if resp, ok, err := rc.cache.Get(key); ok && err == nil {
		if r, ok := resp.(*cachedResponse); ok {
			return r
		}
		log.Errorf("could not parse cached response %v for key %s", resp, key)
	} else if err != nil {
		log.Debugf("could not read cache value for [ key = %v , err = %v ]", key, err)
	}
	return nil
}

// extractCacheHeaders extracts the client request headers allowing the client some control over the cache
func extractCacheHeaders(header string, minAge, maxFresh int64) (*cacheControl, string) {
	if header == "" {
		return &cacheControl{noCache: minAge == 0 && maxFresh == 0}, ""
	}

	cfg := cacheControl{
		validators: make([]validator, 0),
	}

	var warning string
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
			cfg.noCache = true
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
		warning = strings.Join(wrn, ",")
	}

	return &cfg, warning
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

type cachedResponse struct {
	response  *cacheHandlerResponse
	lastValid int64
	etag      string
	err       error
}

func extractRequestKey(path, query string) string {
	return fmt.Sprintf("%s:%s", path, query)
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
