package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/beatlabs/patron/cache"

	"github.com/beatlabs/patron/component/http/auth"
	errs "github.com/beatlabs/patron/errors"
)

// Route definition of a HTTP route.
type Route struct {
	path        string
	method      string
	handler     http.HandlerFunc
	middlewares []MiddlewareFunc
}

// RouteBuilder for building a route.
type RouteBuilder struct {
	method        string
	path          string
	trace         bool
	middlewares   []MiddlewareFunc
	authenticator auth.Authenticator
	handler       http.HandlerFunc
	errors        []error
}

// WithTrace enables route tracing.
func (rb *RouteBuilder) WithTrace() *RouteBuilder {
	rb.trace = true
	return rb
}

// WithMiddlewares adds middlewares.
func (rb *RouteBuilder) WithMiddlewares(mm ...MiddlewareFunc) *RouteBuilder {
	if len(mm) == 0 {
		rb.errors = append(rb.errors, errors.New("middlewares are empty"))
	}
	rb.middlewares = mm
	return rb
}

// WithAuth adds authenticator.
func (rb *RouteBuilder) WithAuth(auth auth.Authenticator) *RouteBuilder {
	if auth == nil {
		rb.errors = append(rb.errors, errors.New("authenticator is nil"))
	}
	rb.authenticator = auth
	return rb
}

func (rb *RouteBuilder) setMethod(method string) *RouteBuilder {
	if rb.method != "" {
		rb.errors = append(rb.errors, errors.New("method already set"))
	}
	rb.method = method
	return rb
}

// MethodGet HTTP method.
func (rb *RouteBuilder) MethodGet() *RouteBuilder {
	return rb.setMethod(http.MethodGet)
}

// MethodHead HTTP method.
func (rb *RouteBuilder) MethodHead() *RouteBuilder {
	return rb.setMethod(http.MethodHead)
}

// MethodPost HTTP method.
func (rb *RouteBuilder) MethodPost() *RouteBuilder {
	return rb.setMethod(http.MethodPost)
}

// MethodPut HTTP method.
func (rb *RouteBuilder) MethodPut() *RouteBuilder {
	return rb.setMethod(http.MethodPut)
}

// MethodPatch HTTP method.
func (rb *RouteBuilder) MethodPatch() *RouteBuilder {
	return rb.setMethod(http.MethodPatch)
}

// MethodDelete HTTP method.
func (rb *RouteBuilder) MethodDelete() *RouteBuilder {
	return rb.setMethod(http.MethodDelete)
}

// MethodConnect HTTP method.
func (rb *RouteBuilder) MethodConnect() *RouteBuilder {
	return rb.setMethod(http.MethodConnect)
}

// MethodOptions HTTP method.
func (rb *RouteBuilder) MethodOptions() *RouteBuilder {
	return rb.setMethod(http.MethodOptions)
}

// MethodTrace HTTP method.
func (rb *RouteBuilder) MethodTrace() *RouteBuilder {
	return rb.setMethod(http.MethodTrace)
}

// Build a route.
func (rb *RouteBuilder) Build() (Route, error) {
	if len(rb.errors) > 0 {
		return Route{}, errs.Aggregate(rb.errors...)
	}

	if rb.method == "" {
		return Route{}, errors.New("method is missing")
	}

	var middlewares []MiddlewareFunc
	if rb.trace {
		middlewares = append(middlewares, NewLoggingTracingMiddleware(rb.path))
	}
	if rb.authenticator != nil {
		middlewares = append(middlewares, NewAuthMiddleware(rb.authenticator))
	}
	if len(rb.middlewares) > 0 {
		middlewares = append(middlewares, rb.middlewares...)
	}

	return Route{
		path:        rb.path,
		method:      rb.method,
		handler:     rb.handler,
		middlewares: middlewares,
	}, nil
}

// NewRawRouteBuilder constructor.
func NewRawRouteBuilder(path string, handler http.HandlerFunc) *RouteBuilder {
	var ee []error

	if path == "" {
		ee = append(ee, errors.New("path is empty"))
	}

	if handler == nil {
		ee = append(ee, errors.New("handler is nil"))
	}

	return &RouteBuilder{path: path, errors: ee, handler: handler}
}

// NewRouteBuilder constructor.
func NewRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {

	var err error

	if processor == nil {
		err = errors.New("processor is nil")
	}

	rb := NewRawRouteBuilder(path, handler(processor))
	if err != nil {
		rb.errors = append(rb.errors, err)
	}
	return rb
}

// RoutesBuilder creates a list of routes.
type RoutesBuilder struct {
	routes []Route
	errors []error
}

// Append a route to the list.
func (rb *RoutesBuilder) Append(builder *RouteBuilder) *RoutesBuilder {
	route, err := builder.Build()
	if err != nil {
		rb.errors = append(rb.errors, err)
	} else {
		rb.routes = append(rb.routes, route)
	}
	return rb
}

// Build the routes.
func (rb *RoutesBuilder) Build() ([]Route, error) {

	duplicates := make(map[string]struct{}, len(rb.routes))

	for _, r := range rb.routes {
		key := strings.ToLower(r.method + "-" + r.path)
		_, ok := duplicates[key]
		if ok {
			rb.errors = append(rb.errors, fmt.Errorf("route with key %s is duplicate", key))
			continue
		}
		duplicates[key] = struct{}{}
	}

	if len(rb.errors) > 0 {
		return nil, errs.Aggregate(rb.errors...)
	}

	return rb.routes, nil
}

// NewRoutesBuilder constructor.
func NewRoutesBuilder() *RoutesBuilder {
	return &RoutesBuilder{}
}

type CachedRouteBuilder struct {
	path          string
	processor     sync.ProcessorFunc
	cache         cache.Cache
	instant       TimeInstant
	ttl           time.Duration
	minAge        uint
	maxFresh      uint
	staleResponse bool
	errors        []error
}

// WithTimeInstant specifies a time instant function for checking expiry.
func (cb *CachedRouteBuilder) WithTimeInstant(instant TimeInstant) *CachedRouteBuilder {
	if instant == nil {
		cb.errors = append(cb.errors, errors.New("time instant is nil"))
	}
	cb.instant = instant
	return cb
}

// WithTimeInstant adds a time to live parameter to control the cache expiry policy.
func (cb *CachedRouteBuilder) WithTimeToLive(ttl time.Duration) *CachedRouteBuilder {
	if ttl <= 0 {
		cb.errors = append(cb.errors, errors.New("time to live must be greater than `0`"))
	}
	cb.ttl = ttl
	return cb
}

// WithMinAge adds a minimum age for the cache responses.
// This will avoid cases where a single client with high request rate and no cache control headers might effectively disable the cache
// This means that if this parameter is missing (e.g. is equal to '0' , the cache can effectively be made obsolete in the above scenario)
func (cb *CachedRouteBuilder) WithMinAge(minAge uint) *CachedRouteBuilder {
	cb.minAge = minAge
	return cb
}

// WithMinFresh adds a minimum age for the cache responses.
// This will avoid cases where a single client with high request rate and no cache control headers might effectively disable the cache
// This means that if this parameter is missing (e.g. is equal to '0' , the cache can effectively be made obsolete in the above scenario)
func (cb *CachedRouteBuilder) WithMaxFresh(maxFresh uint) *CachedRouteBuilder {
	cb.maxFresh = maxFresh
	return cb
}

// WithStaleResponse allows the cache to return stale responses.
func (cb *CachedRouteBuilder) WithStaleResponse(staleResponse bool) *CachedRouteBuilder {
	cb.staleResponse = staleResponse
	return cb
}

func (cb *CachedRouteBuilder) Create() (*routeCache, error) {
	//if len(cb.errors) > 0 {
	//ttl > 0
	//maxfresh < ttl
	return &routeCache{}, nil
	//}
}

func NewRouteCache(path string, processor sync.ProcessorFunc, cache cache.Cache) *routeCache {
	if strings.ReplaceAll(path, " ", "") == "" {

	}
	return &routeCache{
		path:      path,
		processor: processor,
		cache:     cache,
		instant: func() int64 {
			return time.Now().Unix()
		},
	}
}

// ToGetRouteBuilder transforms the cached builder to a GET endpoint builder
// while propagating any errors
func (cb *CachedRouteBuilder) ToGetRouteBuilder() *RouteBuilder {
	routeCache, err := cb.Create()
	if err == nil {

	}
	rb := NewRouteBuilder(cb.path, cacheHandler(cb.processor, routeCache)).MethodGet()
	rb.errors = append(rb.errors, cb.errors...)
	return rb
}

type routeCache struct {
	// path is the route path, which the cache is enabled for
	path string
	// processor is the processor function for the route
	processor sync.ProcessorFunc
	// cache is the cache implementation to be used
	cache cache.Cache
	// ttl is the time to live for all cached objects
	ttl time.Duration
	// instant is the timing function for the cache expiry calculations
	instant TimeInstant
	// minAge specifies the minimum amount of max-age header value for client cache-control requests
	minAge uint
	// max-fresh specifies the maximum amount of min-fresh header value for client cache-control requests
	maxFresh uint
	// staleResponse specifies if the server is willing to send stale responses
	// if a new response could not be generated for any reason
	staleResponse bool
}
