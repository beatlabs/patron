package http

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/beatlabs/patron/cache"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/sync"
)

type CacheHeader int

const (
	max_age CacheHeader = iota + 1
	max_stale
	min_fresh
	no_cache
	no_store
	no_transform
	only_if_cached

	CacheControlHeader = "CACHE-CONTROL"
)

var cacheHeaders = map[string]CacheHeader{"max-age": max_age, "max-stale": max_stale, "min-fresh": min_fresh, "no-cache": no_cache, "no-store": no_store, "no-transform": no_transform, "only-if-cached": only_if_cached}

type TimeInstant func() int64

func cacheHandler(hnd sync.ProcessorFunc, cache cache.Cache, instant TimeInstant) sync.ProcessorFunc {
	return func(ctx context.Context, request *sync.Request) (response *sync.Response, e error) {

		now := instant()

		skipCache, onlyIfCached, ttl := extractCacheHeaders(request)

		if skipCache {
			return hnd(ctx, request)
		}

		// TODO : add metrics

		key := createRequestKey(request)
		if resp, ok, err := cache.Get(key); ok && err == nil {
			// TODO : cache also errors ???
			if r, ok := resp.(cachedResponse); ok && notExpired(now, r.lastValid, ttl) {
				println(fmt.Sprintf("cache = %v", cache))
				return r.response, r.err
			} else {
				log.Errorf("could not parse cached response from %v", resp)
			}
		} else if err != nil {
			log.Debugf("could not read cache value for [ key = %v , err = %v ]", key, err)
		}

		if onlyIfCached {
			// return empty response if we have cache-only header present
			return sync.NewResponse([]byte{}), nil
		}

		// we have not encountered this key before
		response, e = hnd(ctx, request)
		resp := cachedResponse{
			response:  response,
			lastValid: now,
			err:       e,
		}
		err := cache.Set(key, resp)
		log.Errorf("could not cache response for request key %s %w", key, err)
		return
	}
}

func extractCacheHeaders(request *sync.Request) (bool, bool, int64) {
	var noCache bool
	var forceCache bool
	var ttl int64
	println(fmt.Sprintf("request = %v", request))
	if CacheControl, ok := request.Headers[CacheControlHeader]; ok {
		// time to live threshold
		for _, header := range strings.Split(CacheControl, ",") {
			println(fmt.Sprintf("header = %v", header))
			keyValue := strings.Split(header, "=")
			println(fmt.Sprintf("keyValue = %v", keyValue))
			if cacheHeader, ok := cacheHeaders[keyValue[0]]; ok {
				switch cacheHeader {
				case max_stale:
					/**
					Indicates that the client is willing to accept a response that has
					exceeded its expiration time. If max-stale is assigned a value,
					then the client is willing to accept a response that has exceeded
					its expiration time by no more than the specified number of
					seconds. If no value is assigned to max-stale, then the client is
					willing to accept a stale response of any age.
					*/
					fallthrough
				case max_age:
					/**
					Indicates that the client is willing to accept a response whose
					age is no greater than the specified time in seconds. Unless max-
					stale directive is also included, the client is not willing to
					accept a stale response.
					*/
					expiration, err := strconv.Atoi(keyValue[1])
					if err == nil {
						ttl -= int64(expiration)
					}
				case min_fresh:
					/**
					Indicates that the client is willing to accept a response whose
					freshness lifetime is no less than its current age plus the
					specified time in seconds. That is, the client wants a response
					that will still be fresh for at least the specified number of
					seconds.
					*/
					freshness, err := strconv.Atoi(keyValue[1])
					if err == nil {
						ttl += int64(freshness)
					}
				case no_cache:
					/**
					retrieve from the store
					it SHOULD NOT include min-fresh, max-stale, or max-age.
					*/
					fallthrough
				case no_store:
					/**
					no storage whatsoever
					*/
					noCache = true
				case no_transform:
					/**
					response should be kept intact
					*/
					// we always use no-transform
				case only_if_cached:
					/**
					return only if is in cache , otherwise 504
					*/
					forceCache = true
				default:
					log.Warnf("unrecognised cache header %s", header)
				}
			}
		}
	} else {
		// we dont have any cache-control headers, so no intention of caching anything
		noCache = true
	}
	return noCache, forceCache, ttl
}

type cachedResponse struct {
	response  *sync.Response
	lastValid int64
	err       error
}

func createRequestKey(request *sync.Request) string {
	return fmt.Sprintf("%s:%s", request.Headers, request.Fields)
}

func notExpired(now, last, ttl int64) bool {
	nilExp := now+ttl <= last
	return nilExp
}
