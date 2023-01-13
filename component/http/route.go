package http

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/beatlabs/patron/cache"
	"github.com/beatlabs/patron/component/http/auth"
	httpcache "github.com/beatlabs/patron/component/http/cache"
	"github.com/beatlabs/patron/component/http/middleware"
	errs "github.com/beatlabs/patron/errors"
	"golang.org/x/time/rate"
)

// Route definition of an HTTP route.
type Route struct {
	path        string
	method      string
	handler     http.HandlerFunc
	middlewares []middleware.Func
}

// Path returns route path value.
func (r Route) Path() string {
	return r.path
}

// Method returns route method value (GET/POST/...).
func (r Route) Method() string {
	return r.method
}

// Middlewares returns route middlewares.
func (r Route) Middlewares() []middleware.Func {
	return r.middlewares
}

// Handler returns route handler function.
func (r Route) Handler() http.HandlerFunc {
	return r.handler
}

// RouteBuilder for building a route.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
type RouteBuilder struct {
	method        string
	path          string
	jaegerTrace   bool
	rateLimiter   *rate.Limiter
	middlewares   []middleware.Func
	authenticator auth.Authenticator
	handler       http.HandlerFunc
	routeCache    *httpcache.RouteCache
	errors        []error
}

// WithTrace enables route tracing that uses Jaeger/OpenTracing.
// It requires Jaeger enabled on the Patron service.
func (rb *RouteBuilder) WithTrace() *RouteBuilder {
	rb.jaegerTrace = true
	return rb
}

// WithRateLimiting enables route rate limiting.
func (rb *RouteBuilder) WithRateLimiting(limit float64, burst int) *RouteBuilder {
	rb.rateLimiter = rate.NewLimiter(rate.Limit(limit), burst)
	return rb
}

// WithMiddlewares adds middlewares.
func (rb *RouteBuilder) WithMiddlewares(mm ...middleware.Func) *RouteBuilder {
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

// WithRouteCache adds a cache to the corresponding route.
func (rb *RouteBuilder) WithRouteCache(cache cache.TTLCache, ageBounds httpcache.Age) *RouteBuilder {
	rc, ee := httpcache.NewRouteCache(cache, ageBounds)

	rb.routeCache = rc
	rb.errors = append(rb.errors, ee...)
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
	// parse a list of HTTP numeric status codes that must be logged
	cfg, _ := os.LookupEnv("PATRON_HTTP_STATUS_ERROR_LOGGING")
	statusCodeLogger, err := middleware.NewStatusCodeLoggerHandler(cfg)
	if err != nil {
		return Route{}, fmt.Errorf("failed to parse status codes %q: %w", cfg, err)
	}

	if len(rb.errors) > 0 {
		return Route{}, errs.Aggregate(rb.errors...)
	}

	if rb.method == "" {
		return Route{}, errors.New("method is missing")
	}

	var middlewares []middleware.Func
	if rb.jaegerTrace {
		// uses Jaeger/OpenTracing and Patron's response logging
		loggingTracingMiddleware, err := middleware.NewLoggingTracing(rb.path, statusCodeLogger)
		if err != nil {
			return Route{}, err
		}
		middlewares = append(middlewares, loggingTracingMiddleware)
	}

	// uses a custom Patron metric for HTTP responses (with complete status code)
	// it does not use Jaeger/OpenTracing
	requestObserverMiddleware, err := middleware.NewRequestObserver(rb.method, rb.path)
	if err != nil {
		return Route{}, err
	}
	middlewares = append(middlewares, requestObserverMiddleware)

	if rb.rateLimiter != nil {
		rateLimiterMiddleware, err := middleware.NewRateLimiting(rb.rateLimiter)
		if err != nil {
			return Route{}, err
		}
		middlewares = append(middlewares, rateLimiterMiddleware)
	}
	if rb.authenticator != nil {
		middlewares = append(middlewares, middleware.NewAuth(rb.authenticator))
	}
	if len(rb.middlewares) > 0 {
		middlewares = append(middlewares, rb.middlewares...)
	}
	// cache middleware is always last, so that it caches only the headers of the handler
	if rb.routeCache != nil {
		if rb.method != http.MethodGet {
			return Route{}, errors.New("cannot apply cache to a route with any method other than GET ")
		}
		cachingMiddleware, err := middleware.NewCaching(rb.routeCache)
		if err != nil {
			return Route{}, err
		}
		middlewares = append(middlewares, cachingMiddleware)
	}

	return Route{
		path:        rb.path,
		method:      rb.method,
		handler:     rb.handler,
		middlewares: middlewares,
	}, nil
}

// NewFileServer constructor.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewFileServer(path string, assetsDir string, fallbackPath string) *RouteBuilder {
	var ee []error

	if path == "" {
		ee = append(ee, errors.New("path is empty"))
	}

	if assetsDir == "" {
		ee = append(ee, errors.New("assets path is empty"))
	} else {
		_, err := os.Stat(assetsDir)
		if os.IsNotExist(err) {
			ee = append(ee, fmt.Errorf("assets directory [%s] doesn't exist", path))
		} else if err != nil {
			ee = append(ee, fmt.Errorf("error while checking assets dir: %w", err))
		}
	}

	if fallbackPath == "" {
		ee = append(ee, errors.New("fallback path is empty"))
	} else {
		_, err := os.Stat(fallbackPath)
		if os.IsNotExist(err) {
			ee = append(ee, fmt.Errorf("fallback file [%s] doesn't exist", fallbackPath))
		} else if err != nil {
			ee = append(ee, fmt.Errorf("error while checking fallback file: %w", err))
		}
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		params := ExtractParams(r)

		// get the absolute path to prevent directory traversal
		path := fmt.Sprintf("%s%s", assetsDir, params["path"])

		// check whether a file exists at the given path
		info, err := os.Stat(path)
		if os.IsNotExist(err) || info.IsDir() {
			// file does not exist, serve index.html
			http.ServeFile(w, r, fallbackPath)
			return
		} else if err != nil {
			// if we got an error (that wasn't that the file doesn't exist) stating the
			// file, return a 500 internal server error and stop
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		// otherwise, use server the specific file directly from the filesystem.
		http.ServeFile(w, r, path)
	}

	return &RouteBuilder{path: path, errors: ee, handler: handler, method: http.MethodGet}
}

// NewRawRouteBuilder constructor.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
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
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	var ee []error

	if path == "" {
		ee = append(ee, errors.New("path is empty"))
	}

	if processor == nil {
		ee = append(ee, errors.New("processor is nil"))
	}

	return &RouteBuilder{path: path, errors: ee, handler: handler(processor)}
}

// NewGetRouteBuilder constructor
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewGetRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	return NewRouteBuilder(path, processor).MethodGet()
}

// NewHeadRouteBuilder constructor.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewHeadRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	return NewRouteBuilder(path, processor).MethodHead()
}

// NewPostRouteBuilder constructor.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewPostRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	return NewRouteBuilder(path, processor).MethodPost()
}

// NewPutRouteBuilder constructor.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewPutRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	return NewRouteBuilder(path, processor).MethodPut()
}

// NewPatchRouteBuilder constructor.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewPatchRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	return NewRouteBuilder(path, processor).MethodPatch()
}

// NewDeleteRouteBuilder constructor.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewDeleteRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	return NewRouteBuilder(path, processor).MethodDelete()
}

// NewConnectRouteBuilder constructor.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewConnectRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	return NewRouteBuilder(path, processor).MethodConnect()
}

// NewOptionsRouteBuilder constructor.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewOptionsRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	return NewRouteBuilder(path, processor).MethodOptions()
}

// NewTraceRouteBuilder constructor.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewTraceRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {
	return NewRouteBuilder(path, processor).MethodTrace()
}

// RoutesBuilder creates a list of routes.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
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
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewRoutesBuilder() *RoutesBuilder {
	return &RoutesBuilder{}
}
