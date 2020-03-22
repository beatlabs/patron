package http

import (
	"context"
	"fmt"
	"hash/crc32"
	"strconv"
	"strings"
	"time"

	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/sync"
)

// TODO : add comments where applicable

type CacheHeader int

const (
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
type validator func(age, ttl int64) bool

// expiryCheck is the main validator that checks that the entry has not expired e.g. is stale
var expiryCheck validator = func(age, ttl int64) bool {
	return age <= ttl
}

// cacheControl is the model of the request parameters regarding the cache control
type cacheControl struct {
	noCache         bool
	forceCache      bool
	validators      []validator
	expiryValidator validator
}

// cacheHandler wraps the handler func with a cache layer
// hnd is the processor func that the cache will wrap
// rc is the route cache implementation to be used
func cacheHandler(hnd sync.ProcessorFunc, rc *routeCache) sync.ProcessorFunc {

	return func(ctx context.Context, request *sync.Request) (response *sync.Response, e error) {

		responseHeaders := make(map[string]string)

		now := rc.instant()

		cfg, warning := extractCacheHeaders(request, rc.minAge, rc.maxFresh)
		if cfg.expiryValidator == nil {
			cfg.expiryValidator = expiryCheck
		}
		key := extractRequestKey(rc.path, request)

		// TODO : add metrics

		var rsp *cachedResponse
		var fromCache bool

		// explore the cache
		if cfg.noCache && !rc.staleResponse {
			// need to execute the handler always
			rsp = handlerExecutor(ctx, request, hnd, now, key)
		} else {
			// lets check the cache if we have anything for the given key
			if rsp = cacheRetriever(key, rc, now); rsp == nil {
				// we have not encountered this key before
				rsp = handlerExecutor(ctx, request, hnd, now, key)
			} else {
				expiry := int64(rc.ttl / time.Second)
				if !isValid(expiry, now, rsp.lastValid, append(cfg.validators, cfg.expiryValidator)...) {
					tmpRsp := handlerExecutor(ctx, request, hnd, now, key)
					if rc.staleResponse && tmpRsp.err != nil {
						warning = "last-valid"
						fromCache = true
					} else {
						rsp = tmpRsp
					}
				} else {
					fromCache = true
				}
			}
		}
		// TODO : use the forceCache parameter
		if cfg.forceCache {
			// return empty response if we have rc-only responseHeaders present
			return sync.NewResponse([]byte{}), nil
		}

		response = rsp.response
		e = rsp.err

		// TODO : abstract into method
		if e == nil {
			responseHeaders[eTagHeader] = rsp.etag

			responseHeaders[cacheControlHeader] = genCacheControlHeader(rc.ttl, now-rsp.lastValid)

			if warning != "" && fromCache {
				responseHeaders[warningHeader] = warning
			}

			response.Headers = responseHeaders
		}

		// we cache response only if we did not retrieve it from the cache itself and error is nil
		if !fromCache && e == nil {
			if err := rc.cache.Set(key, rsp); err != nil {
				log.Errorf("could not cache response for request key %s %v", key, err)
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
		} else {
			log.Errorf("could not parse cached response %v for key %s", resp, key)
		}
	} else if err != nil {
		log.Debugf("could not read cache value for [ key = %v , err = %v ]", key, err)
	}
	return nil
}

// handlerExecutor is the function that will create a new cachedResponse from based on the handler implementation
var handlerExecutor = func(ctx context.Context, request *sync.Request, hnd sync.ProcessorFunc, now int64, key string) *cachedResponse {
	response, err := hnd(ctx, request)
	return &cachedResponse{
		response:  response,
		lastValid: now,
		etag:      genETag([]byte(key), time.Now().Nanosecond()),
		err:       err,
	}
}

// extractCacheHeaders extracts the client request headers allowing the client some control over the cache
func extractCacheHeaders(request *sync.Request, minAge, maxFresh uint) (*cacheControl, string) {
	if CacheControl, ok := request.Headers[cacheControlHeader]; ok {
		return extractCacheHeader(CacheControl, minAge, maxFresh)
	}
	// if we have no headers we assume we dont want to cache,
	return &cacheControl{noCache: minAge == 0 && maxFresh == 0}, ""
}

func extractCacheHeader(headers string, minAge, maxFresh uint) (*cacheControl, string) {
	cfg := cacheControl{
		validators: make([]validator, 0),
	}

	var warning string
	wrn := make([]string, 0)

	for _, header := range strings.Split(headers, ",") {
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
			cfg.expiryValidator = func(age, ttl int64) bool {
				return ttl-age+value >= 0
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
			value, adjusted := min(value, int64(minAge))
			if adjusted {
				wrn = append(wrn, fmt.Sprintf("max-age=%d", minAge))
			}
			cfg.validators = append(cfg.validators, func(age, ttl int64) bool {
				return age <= value
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
			value, adjusted := max(value, int64(maxFresh))
			if adjusted {
				wrn = append(wrn, fmt.Sprintf("min-fresh=%d", maxFresh))
			}
			cfg.validators = append(cfg.validators, func(age, ttl int64) bool {
				return ttl-age >= value
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
	response  *sync.Response
	lastValid int64
	etag      string
	err       error
}

func extractRequestKey(path string, request *sync.Request) string {
	return fmt.Sprintf("%s:%s", path, request.Fields)
}

func isValid(ttl, now, last int64, validators ...validator) bool {
	if len(validators) == 0 {
		return false
	}
	age := now - last
	for _, validator := range validators {
		if !validator(age, ttl) {
			return false
		}
	}
	return true
}

func genETag(key []byte, t int) string {
	return fmt.Sprintf("%d-%d", crc32.ChecksumIEEE(key), t)
}

func genCacheControlHeader(ttl time.Duration, lastValid int64) string {
	maxAge := int64(ttl/time.Second) - lastValid
	if maxAge <= 0 {
		return fmt.Sprintf("%s", mustRevalidate)
	}
	return fmt.Sprintf("%s=%d", maxAge, int64(ttl/time.Second)-lastValid)
}
