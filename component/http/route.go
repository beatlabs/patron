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
	errs "github.com/beatlabs/patron/errors"
	"golang.org/x/time/rate"
)

// Route definition of a HTTP route.
type Route struct {
	path        string
	method      string
	handler     http.HandlerFunc
	middlewares []MiddlewareFunc
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
func (r Route) Middlewares() []MiddlewareFunc {
	return r.middlewares
}

// Handler returns route handler function.
func (r Route) Handler() http.HandlerFunc {
	return r.handler
}

// RouteBuilder for building a route.
type RouteBuilder struct {
	method        string
	path          string
	trace         bool
	rateLimiter   *rate.Limiter
	middlewares   []MiddlewareFunc
	authenticator auth.Authenticator
	handler       http.HandlerFunc
	routeCache    *httpcache.RouteCache
	errors        []error
}

// WithTrace enables route tracing.
func (rb *RouteBuilder) WithTrace() *RouteBuilder {
	rb.trace = true
	return rb
}

// WithRateLimiting enables route rate limiting.
func (rb *RouteBuilder) WithRateLimiting(limit float64, burst int) *RouteBuilder {
	rb.rateLimiter = rate.NewLimiter(rate.Limit(limit), burst)
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

// WithRouteCache adds a cache to the corresponding route
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
	cfg, _ := os.LookupEnv("PATRON_HTTP_STATUS_ERROR_LOGGING")
	statusCodeLogger, err := newStatusCodeLoggerHandler(cfg)
	if err != nil {
		return Route{}, fmt.Errorf("failed to parse status codes %q: %w", cfg, err)
	}

	if len(rb.errors) > 0 {
		return Route{}, errs.Aggregate(rb.errors...)
	}

	if rb.method == "" {
		return Route{}, errors.New("method is missing")
	}

	var middlewares []MiddlewareFunc
	if rb.trace {
		middlewares = append(middlewares, NewLoggingTracingMiddleware(rb.path, statusCodeLogger))
	}
	if rb.rateLimiter != nil {
		middlewares = append(middlewares, NewRateLimitingMiddleware(rb.rateLimiter))
	}
	if rb.authenticator != nil {
		middlewares = append(middlewares, NewAuthMiddleware(rb.authenticator))
	}
	if len(rb.middlewares) > 0 {
		middlewares = append(middlewares, rb.middlewares...)
	}
	// cache middleware is always last, so that it caches only the headers of the handler
	if rb.routeCache != nil {
		if rb.method != http.MethodGet {
			return Route{}, errors.New("cannot apply cache to a route with any method other than GET ")
		}
		middlewares = append(middlewares, NewCachingMiddleware(rb.routeCache))
	}

	return Route{
		path:        rb.path,
		method:      rb.method,
		handler:     rb.handler,
		middlewares: middlewares,
	}, nil
}

// NewFileServer constructor.
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
func NewGetRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {

	return NewRouteBuilder(path, processor).MethodGet()
}

// NewHeadRouteBuilder constructor.
func NewHeadRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {

	return NewRouteBuilder(path, processor).MethodHead()
}

// NewPostRouteBuilder constructor.
func NewPostRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {

	return NewRouteBuilder(path, processor).MethodPost()
}

// NewPutRouteBuilder constructor.
func NewPutRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {

	return NewRouteBuilder(path, processor).MethodPut()
}

// NewPatchRouteBuilder constructor.
func NewPatchRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {

	return NewRouteBuilder(path, processor).MethodPatch()
}

// NewDeleteRouteBuilder constructor.
func NewDeleteRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {

	return NewRouteBuilder(path, processor).MethodDelete()
}

// NewConnectRouteBuilder constructor.
func NewConnectRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {

	return NewRouteBuilder(path, processor).MethodConnect()
}

// NewOptionsRouteBuilder constructor.
func NewOptionsRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {

	return NewRouteBuilder(path, processor).MethodOptions()
}

// NewTraceRouteBuilder constructor.
func NewTraceRouteBuilder(path string, processor ProcessorFunc) *RouteBuilder {

	return NewRouteBuilder(path, processor).MethodTrace()
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
