package http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

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
	method            string
	path              string
	trace             bool
	middlewares       []MiddlewareFunc
	authenticator     auth.Authenticator
	processor         ProcessorFunc
	handler           http.HandlerFunc
	routeCacheBuilder *RouteCacheBuilder
	errors            []error
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

func (rb *RouteBuilder) WithRouteCachedBuilder(routeCacheBuilder *RouteCacheBuilder) *RouteBuilder {
	if routeCacheBuilder == nil {
		rb.errors = append(rb.errors, errors.New("cache route builder is nil"))
	}
	rb.routeCacheBuilder = routeCacheBuilder
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

	// TODO : refactor appropriately
	var processor ProcessorFunc
	var handler http.HandlerFunc

	if rb.routeCacheBuilder != nil {

		if rb.method != http.MethodGet {
			return Route{}, errors.New("cannot apply cache to a route with any method other than GET ")
		}

		rc, err := rb.routeCacheBuilder.create(rb.path)
		if err != nil {
			return Route{}, fmt.Errorf("could not build cache from builder %v: %w", rb.routeCacheBuilder, err)
		}

		// TODO : we need to refactor the abstraction in issue #160
		if rb.processor != nil {
			// builder was initialised from the NewRouteBuilder constructor
			// e.g. the only place where the rb.processor is set
			processor = wrapProcessorFunc(rb.path, rb.processor, rc)
		} else {
			// we could have handled the processor also at a middleware level,
			// but this would not work uniformly for the above case as well.
			handler = wrapHandlerFunc(rb.handler, rc)
		}
	}

	if processor == nil {
		processor = rb.processor
	}

	if handler == nil {
		handler = rb.handler
	}

	return Route{
		path:        rb.path,
		method:      rb.method,
		handler:     constructHTTPHandler(processor, handler),
		middlewares: middlewares,
	}, nil
}

func constructHTTPHandler(processor ProcessorFunc, httpHandler http.HandlerFunc) http.HandlerFunc {
	if processor == nil {
		return httpHandler
	}
	return handler(processor)
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

	return &RouteBuilder{path: path, errors: ee, processor: processor}
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
