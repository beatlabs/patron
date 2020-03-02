package http

import (
	"errors"
	"net/http"

	patronerrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/sync"
	"github.com/beatlabs/patron/sync/http/auth"
)

// Method for HTTP.
type Method string

const (
	// MethodGet for HTTP.
	MethodGet Method = http.MethodGet
	// MethodHead for HTTP.
	MethodHead Method = http.MethodHead
	// MethodPost for HTTP.
	MethodPost Method = http.MethodPost
	// MethodPut for HTTP.
	MethodPut Method = http.MethodPut
	// MethodPatch for HTTP.
	MethodPatch Method = http.MethodPatch
	// MethodDelete for HTTP.
	MethodDelete Method = http.MethodDelete
	// MethodConnect for HTTP.
	MethodConnect Method = http.MethodConnect
	// MethodOptions for HTTP.
	MethodOptions Method = http.MethodOptions
	// MethodTrace for HTTP.
	MethodTrace Method = http.MethodTrace
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
	method        Method
	path          string
	trace         bool
	middlewares   []MiddlewareFunc
	authenticator auth.Authenticator
	handler       http.HandlerFunc
	errors        []error
}

// WithTrace enables route tracing.
func (rb *RouteBuilder) WithTrace() *RouteBuilder {
	if len(rb.errors) > 0 {
		return rb
	}
	rb.trace = true
	return rb
}

// WithMiddlewares adds middlewares.
func (rb *RouteBuilder) WithMiddlewares(mm ...MiddlewareFunc) *RouteBuilder {
	if len(rb.errors) > 0 {
		return rb
	}
	if len(mm) == 0 {
		rb.errors = append(rb.errors, errors.New("middlewares are empty"))
	}
	rb.middlewares = mm
	return rb
}

// WithAuth adds authenticator.
func (rb *RouteBuilder) WithAuth(auth auth.Authenticator) *RouteBuilder {
	if len(rb.errors) > 0 {
		return rb
	}
	if auth == nil {
		rb.errors = append(rb.errors, errors.New("authenticator is nil"))
	}
	rb.authenticator = auth
	return rb
}

// Build a route.
func (rb *RouteBuilder) Build() (Route, error) {
	if len(rb.errors) > 0 {
		return Route{}, patronerrors.Aggregate(rb.errors...)
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
		method:      string(rb.method),
		handler:     rb.handler,
		middlewares: middlewares,
	}, nil
}

// NewRawRouteBuilder constructor.
func NewRawRouteBuilder(method Method, path string, handler http.HandlerFunc) *RouteBuilder {
	var ee []error

	if method == "" {
		ee = append(ee, errors.New("method is empty"))
	}

	if path == "" {
		ee = append(ee, errors.New("path is empty"))
	}

	if handler == nil {
		ee = append(ee, errors.New("handler is nil"))
	}

	return &RouteBuilder{method: method, path: path, errors: ee, handler: handler}
}

// NewRouteBuilder constructor.
func NewRouteBuilder(method Method, path string, processor sync.ProcessorFunc) *RouteBuilder {

	var err error

	if processor == nil {
		err = errors.New("processor is nil")
	}

	rb := NewRawRouteBuilder(method, path, handler(processor))
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
	if len(rb.errors) > 0 {
		return nil, patronerrors.Aggregate(rb.errors...)
	}
	return rb.routes, nil
}

// NewRoutesBuilder constructor.
func NewRoutesBuilder() *RoutesBuilder {
	return &RoutesBuilder{}
}
