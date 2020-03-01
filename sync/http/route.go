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

// WithProcessor adds a processor func.
func (rb *RouteBuilder) WithProcessor(pr sync.ProcessorFunc) *RouteBuilder {
	if len(rb.errors) > 0 {
		return rb
	}
	if pr == nil {
		rb.errors = append(rb.errors, errors.New("processor func is nil"))
	}
	rb.handler = handler(pr)
	return rb
}

// WithRawHandler adds a std http handler func.
func (rb *RouteBuilder) WithRawHandler(hf http.HandlerFunc) *RouteBuilder {
	if len(rb.errors) > 0 {
		return rb
	}
	if hf == nil {
		rb.errors = append(rb.errors, errors.New("raw handler func is nil"))
	}
	rb.handler = hf
	return rb
}

// Build a route.
func (rb *RouteBuilder) Build() (*Route, error) {
	if len(rb.errors) > 0 {
		return nil, patronerrors.Aggregate(rb.errors...)
	}

	if rb.handler == nil {
		return nil, errors.New("handler is nil")
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

	return &Route{
		path:        rb.path,
		method:      string(rb.method),
		handler:     rb.handler,
		middlewares: middlewares,
	}, nil
}

// NewRouteBuilder constructor.
func NewRouteBuilder(method Method, path string) *RouteBuilder {
	var ee []error

	if method == "" {
		ee = append(ee, errors.New("method is empty"))
	}

	if path == "" {
		ee = append(ee, errors.New("path is empty"))
	}

	return &RouteBuilder{method: method, path: path, errors: ee}
}

// Route definition of a HTTP route.
type Route struct {
	path        string
	method      string
	handler     http.HandlerFunc
	middlewares []MiddlewareFunc
}

// NewGetRoute creates a new GET route from a generic handler.
func NewGetRoute(p string, pr sync.ProcessorFunc, trace bool, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodGet, pr, trace, nil, mm...)
}

// NewPostRoute creates a new POST route from a generic handler.
func NewPostRoute(p string, pr sync.ProcessorFunc, trace bool, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodPost, pr, trace, nil, mm...)
}

// NewPutRoute creates a new PUT route from a generic handler.
func NewPutRoute(p string, pr sync.ProcessorFunc, trace bool, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodPut, pr, trace, nil, mm...)
}

// NewDeleteRoute creates a new DELETE route from a generic handler.
func NewDeleteRoute(p string, pr sync.ProcessorFunc, trace bool, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodDelete, pr, trace, nil, mm...)
}

// NewPatchRoute creates a new PATCH route from a generic handler.
func NewPatchRoute(p string, pr sync.ProcessorFunc, trace bool, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodPatch, pr, trace, nil, mm...)
}

// NewHeadRoute creates a new HEAD route from a generic handler.
func NewHeadRoute(p string, pr sync.ProcessorFunc, trace bool, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodHead, pr, trace, nil, mm...)
}

// NewOptionsRoute creates a new OPTIONS route from a generic handler.
func NewOptionsRoute(p string, pr sync.ProcessorFunc, trace bool, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodOptions, pr, trace, nil, mm...)
}

// NewRoute creates a new route from a generic handler with auth capability.
func NewRoute(p string, m string, pr sync.ProcessorFunc, trace bool, auth auth.Authenticator, mm ...MiddlewareFunc) Route {
	var middlewares []MiddlewareFunc
	if trace {
		middlewares = append(middlewares, NewLoggingTracingMiddleware(p))
	}
	if auth != nil {
		middlewares = append(middlewares, NewAuthMiddleware(auth))
	}
	if len(mm) > 0 {
		middlewares = append(middlewares, mm...)
	}
	return Route{path: p, method: m, handler: handler(pr), middlewares: middlewares}
}

// NewRouteRaw creates a new route from a HTTP handler.
func NewRouteRaw(p string, m string, h http.HandlerFunc, trace bool, mm ...MiddlewareFunc) Route {
	var middlewares []MiddlewareFunc
	if trace {
		middlewares = append(middlewares, NewLoggingTracingMiddleware(p))
	}
	if len(mm) > 0 {
		middlewares = append(middlewares, mm...)
	}
	return Route{path: p, method: m, handler: h, middlewares: middlewares}
}

// NewAuthGetRoute creates a new GET route from a generic handler with auth capability.
func NewAuthGetRoute(p string, pr sync.ProcessorFunc, trace bool, auth auth.Authenticator, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodGet, pr, trace, auth, mm...)
}

// NewAuthPostRoute creates a new POST route from a generic handler with auth capability.
func NewAuthPostRoute(p string, pr sync.ProcessorFunc, trace bool, auth auth.Authenticator, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodPost, pr, trace, auth, mm...)
}

// NewAuthPutRoute creates a new PUT route from a generic handler with auth capability.
func NewAuthPutRoute(p string, pr sync.ProcessorFunc, trace bool, auth auth.Authenticator, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodPut, pr, trace, auth, mm...)
}

// NewAuthDeleteRoute creates a new DELETE route from a generic handler with auth capability.
func NewAuthDeleteRoute(p string, pr sync.ProcessorFunc, trace bool, auth auth.Authenticator, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodDelete, pr, trace, auth, mm...)
}

// NewAuthPatchRoute creates a new PATCH route from a generic handler with auth capability.
func NewAuthPatchRoute(p string, pr sync.ProcessorFunc, trace bool, auth auth.Authenticator, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodPatch, pr, trace, auth, mm...)
}

// NewAuthHeadRoute creates a new HEAD route from a generic handler with auth capability.
func NewAuthHeadRoute(p string, pr sync.ProcessorFunc, trace bool, auth auth.Authenticator, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodHead, pr, trace, auth, mm...)
}

// NewAuthOptionsRoute creates a new OPTIONS route from a generic handler with auth capability.
func NewAuthOptionsRoute(p string, pr sync.ProcessorFunc, trace bool, auth auth.Authenticator, mm ...MiddlewareFunc) Route {
	return NewRoute(p, http.MethodOptions, pr, trace, auth, mm...)
}

// NewAuthRouteRaw creates a new route from a HTTP handler with auth capability.
func NewAuthRouteRaw(p string, m string, h http.HandlerFunc, trace bool, auth auth.Authenticator, mm ...MiddlewareFunc) Route {
	var middlewares []MiddlewareFunc
	if trace {
		middlewares = append(middlewares, NewLoggingTracingMiddleware(p))
	}
	if auth != nil {
		middlewares = append(middlewares, NewAuthMiddleware(auth))
	}
	if len(mm) > 0 {
		middlewares = append(middlewares, mm...)
	}
	return Route{path: p, method: m, handler: h, middlewares: middlewares}
}
