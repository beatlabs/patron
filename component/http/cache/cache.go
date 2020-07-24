// Package cache provides a cache control and implementation components for http routes.
package cache

import (
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
	// maxAgeValidation represents a validation , happening due to max-age  Header requirements
	maxAgeValidation
	// minFreshValidation represents a validation , happening due to min-fresh  Header requirements
	minFreshValidation

	// HeaderCacheControl is the Header key for cache related values
	// note : it is case-sensitive
	HeaderCacheControl = "Cache-Control"
	// HeaderETagHeader is the constant representing the Etag http header
	HeaderETagHeader = "Etag"

	controlMinFresh     = "min-fresh"
	controlNoCache      = "no-cache"
	controlNoStore      = "no-store"
	controlOnlyIfCached = "only-if-cached"
	controlEmpty        = ""

	headerCacheMaxAge    = "max-age"
	headerMustRevalidate = "must-revalidate"
	headerWarning        = "Warning"
)

var monitor metrics

func init() {
	monitor = newPrometheusMetrics()
}

// NowSeconds returns the current unix timestamp in seconds
var NowSeconds = func() int64 {
	return time.Now().Unix()
}

// validator is a conditional function on an objects age and the configured ttl
type validator func(age, ttl int64) (bool, validationContext)

// expiryCheck is the main validator that checks that the entry has not expired e.g. is stale
var expiryCheck validator = func(age, ttl int64) (bool, validationContext) {
	return age <= ttl, ttlValidation
}

// control is the model of the request parameters regarding the cache control
type control struct {
	noCache         bool
	forceCache      bool
	warning         string
	validators      []validator
	expiryValidator validator
}

// executor is the function returning a cache Response object from the underlying implementation
type executor func(now int64, key string) *response

// handler wraps the an execution logic with a cache layer
// exec is the processor func that the cache will wrap
// rc is the route cache implementation to be used
func handler(exec executor, rc *RouteCache) func(request *handlerRequest) (response *handlerResponse, e error) {

	return func(request *handlerRequest) (handlerResponse *handlerResponse, e error) {

		now := NowSeconds()

		key := request.getKey()

		var rsp *response

		if hasNoAgeConfig(rc.age.min, rc.age.max) {
			rsp = exec(now, key)
			return &rsp.Response, rsp.Err
		}

		cfg := extractRequestHeaders(request.header, rc.age.min, rc.age.max-rc.age.min)
		if cfg.expiryValidator == nil {
			cfg.expiryValidator = expiryCheck
		}

		rsp = getResponse(cfg, request.path, key, now, rc, exec)
		e = rsp.Err

		if e == nil {
			handlerResponse = &rsp.Response
			addResponseHeaders(now, handlerResponse.Header, rsp, rc.age.max)
			if !rsp.FromCache && !cfg.noCache {
				save(request.path, key, rsp, rc.cache, time.Duration(rc.age.max)*time.Second)
			}
		}

		return
	}
}

// getResponse will get the appropriate Response either using the cache or the executor,
// depending on the
func getResponse(cfg *control, path, key string, now int64, rc *RouteCache, exec executor) *response {

	if cfg.noCache {
		return exec(now, key)
	}

	rsp := get(key, rc)
	if rsp == nil {
		monitor.miss(path)
		response := exec(now, key)
		return response
	}
	if rsp.Err != nil {
		log.Errorf("error during cache interaction: %v", rsp.Err)
		monitor.err(path)
		return exec(now, key)
	}
	// if the object has expired
	if isValid, cx := isValid(now-rsp.LastValid, rc.age.max, append(cfg.validators, cfg.expiryValidator)...); !isValid {
		tmpRsp := exec(now, key)
		// if we could not retrieve a fresh Response,
		// serve the last cached value, with a Warning Header
		if cfg.forceCache || tmpRsp.Err != nil {
			rsp.Warning = "last-valid"
			monitor.hit(path)
		} else {
			rsp = tmpRsp
			monitor.evict(path, cx, now-rsp.LastValid)
		}
	} else {
		// add any Warning generated while parsing the headers
		rsp.Warning = cfg.warning
		monitor.hit(path)
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

// get is the implementation that will provide a response instance from the cache,
// if it exists
func get(key string, rc *RouteCache) *response {
	if resp, ok, err := rc.cache.Get(key); ok && err == nil {
		if b, ok := resp.([]byte); ok {
			r := &response{}
			err := r.decode(b)
			if err != nil {
				return &response{Err: fmt.Errorf("could not decode cached bytes as response %v for key %s", resp, key)}
			}
			r.FromCache = true
			return r
		}
		// NOTE : we need to do this hack to bypass the redis go client implementation of returning result as string instead of bytes
		if b, ok := resp.(string); ok {
			r := &response{}
			err := r.decode([]byte(b))
			if err != nil {
				return &response{Err: fmt.Errorf("could not decode cached string as response %v for key %s", resp, key)}
			}
			r.FromCache = true
			return r
		}

		return &response{Err: fmt.Errorf("could not parse cached response %v for key %s", resp, key)}
	} else if err != nil {
		return &response{Err: fmt.Errorf("could not read cache value for [ key = %v , Err = %v ]", key, err)}
	}
	return nil
}

// save caches the given Response if required with a ttl
// as we are putting the objects in the cache, if its a TTL one, we need to manage the expiration on our own
func save(path, key string, rsp *response, cache cache.TTLCache, maxAge time.Duration) {
	if !rsp.FromCache && rsp.Err == nil {
		// encode to a byte array on our side to avoid cache specific encoding / marshaling requirements
		bytes, err := rsp.encode()
		if err != nil {
			log.Errorf("could not encode response for request key %s: %v", key, err)
			monitor.err(path)
			return
		}
		if err := cache.SetTTL(key, bytes, maxAge); err != nil {
			log.Errorf("could not cache response for request key %s: %v", key, err)
			monitor.err(path)
			return
		}
		monitor.add(path)
	}
}

// addResponseHeaders adds the appropriate headers according to the response conditions
func addResponseHeaders(now int64, header http.Header, rsp *response, maxAge int64) {
	header.Set(HeaderETagHeader, rsp.Etag)
	header.Set(HeaderCacheControl, createCacheControlHeader(maxAge, now-rsp.LastValid))
	if rsp.Warning != "" && rsp.FromCache {
		header.Set(headerWarning, rsp.Warning)
	} else {
		delete(header, headerWarning)
	}
}

// extractRequestHeaders extracts the client request headers allowing the client some control over the cache
func extractRequestHeaders(header string, minAge, maxFresh int64) *control {

	cfg := control{
		validators: make([]validator, 0),
	}

	wrn := make([]string, 0)

	for _, header := range strings.Split(header, ",") {
		keyValue := strings.Split(header, "=")
		headerKey := strings.ToLower(keyValue[0])
		switch headerKey {
		case headerCacheMaxAge:
			/**
			Indicates that the client is willing to accept a Response whose
			age is no greater than the specified time in seconds. Unless max-
			stale directive is also included, the client is not willing to
			accept a stale Response.
			*/
			value, ok := parseValue(keyValue)
			if !ok || value < 0 {
				log.Debugf("invalid value for Header '%s', defaulting to '0' ", keyValue)
				value = 0
			}
			value, adjusted := min(value, minAge)
			if adjusted {
				wrn = append(wrn, fmt.Sprintf("max-age=%d", minAge))
			}
			cfg.validators = append(cfg.validators, func(age, maxAge int64) (bool, validationContext) {
				return age <= value, maxAgeValidation
			})
		case controlMinFresh:
			/**
			Indicates that the client is willing to accept a Response whose
			freshness lifetime is no less than its current age plus the
			specified time in seconds. That is, the client wants a Response
			that will still be fresh for at least the specified number of
			seconds.
			*/
			value, ok := parseValue(keyValue)
			if !ok || value < 0 {
				log.Debugf("invalid value for Header '%s', defaulting to '0' ", keyValue)
				value = 0
			}
			value, adjusted := max(value, maxFresh)
			if adjusted {
				wrn = append(wrn, fmt.Sprintf("min-fresh=%d", maxFresh))
			}
			cfg.validators = append(cfg.validators, func(age, maxAge int64) (bool, validationContext) {
				return maxAge-age >= value, minFreshValidation
			})
		case controlNoCache:
			/**
			return Response if entity has changed
			e.g. (304 Response if nothing has changed : 304 Not Modified)
			it SHOULD NOT include min-fresh, max-stale, or max-age.
			request should be accompanied by an ETag token
			*/
			fallthrough
		case controlNoStore:
			/**
			no storage whatsoever
			*/
			wrn = append(wrn, fmt.Sprintf("max-age=%d", minAge))
			cfg.validators = append(cfg.validators, func(age, maxAge int64) (bool, validationContext) {
				return age <= minAge, maxAgeValidation
			})
		case controlOnlyIfCached:
			/**
			return only if is in cache , otherwise 504
			*/
			cfg.forceCache = true
		case controlEmpty:
			// nothing to do here
		default:
			log.Warn("unrecognised cache Header: '%s'", header)
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
		return headerMustRevalidate
	}
	return fmt.Sprintf("%s=%d", headerCacheMaxAge, ttl-lastValid)
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
