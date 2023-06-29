package http

import (
	"errors"
	"net/http"

	patronhttp "github.com/beatlabs/patron/component/http/middleware"
	patronerrors "github.com/beatlabs/patron/errors"
)

// RouteOptionFunc definition for configuring the route in a functional way.
type RouteOptionFunc func(route *Route) error

// Route definition of an HTTP route.
type Route struct {
	method      string
	path        string
	handler     http.HandlerFunc
	middlewares []patronhttp.Func
}

func (r Route) Method() string {
	return r.method
}

func (r Route) Path() string {
	return r.path
}

func (r Route) Handler() http.HandlerFunc {
	return r.handler
}

func (r Route) Middlewares() []patronhttp.Func {
	return r.middlewares
}

func (r Route) String() string {
	return r.method + " " + r.path
}

// NewRoute creates a new raw route with functional configuration.
func NewRoute(method, path string, handler http.HandlerFunc, oo ...RouteOptionFunc) (*Route, error) {
	if method == "" {
		return nil, errors.New("method is empty")
	}

	if path == "" {
		return nil, errors.New("path is empty")
	}

	if handler == nil {
		return nil, errors.New("handler is nil")
	}

	route := &Route{
		method:  method,
		path:    path,
		handler: handler,
	}

	for _, option := range oo {
		err := option(route)
		if err != nil {
			return nil, err
		}
	}

	return route, nil
}

// NewGetRoute constructor.
func NewGetRoute(path string, handler http.HandlerFunc, oo ...RouteOptionFunc) (*Route, error) {
	return NewRoute(http.MethodGet, path, handler, oo...)
}

// NewHeadRoute constructor.
func NewHeadRoute(path string, handler http.HandlerFunc, oo ...RouteOptionFunc) (*Route, error) {
	return NewRoute(http.MethodHead, path, handler, oo...)
}

// NewPostRoute constructor.
func NewPostRoute(path string, handler http.HandlerFunc, oo ...RouteOptionFunc) (*Route, error) {
	return NewRoute(http.MethodPost, path, handler, oo...)
}

// NewPutRoute constructor.
func NewPutRoute(path string, handler http.HandlerFunc, oo ...RouteOptionFunc) (*Route, error) {
	return NewRoute(http.MethodPut, path, handler, oo...)
}

// NewPatchRoute constructor.
func NewPatchRoute(path string, handler http.HandlerFunc, oo ...RouteOptionFunc) (*Route, error) {
	return NewRoute(http.MethodPatch, path, handler, oo...)
}

// NewDeleteRoute constructor.
func NewDeleteRoute(path string, handler http.HandlerFunc, oo ...RouteOptionFunc) (*Route, error) {
	return NewRoute(http.MethodDelete, path, handler, oo...)
}

// NewConnectRoute constructor.
func NewConnectRoute(path string, handler http.HandlerFunc, oo ...RouteOptionFunc) (*Route, error) {
	return NewRoute(http.MethodConnect, path, handler, oo...)
}

// NewOptionsRoute constructor.
func NewOptionsRoute(path string, handler http.HandlerFunc, oo ...RouteOptionFunc) (*Route, error) {
	return NewRoute(http.MethodOptions, path, handler, oo...)
}

// NewTraceRoute constructor.
func NewTraceRoute(path string, handler http.HandlerFunc, oo ...RouteOptionFunc) (*Route, error) {
	return NewRoute(http.MethodTrace, path, handler, oo...)
}

// Routes definition.
type Routes struct {
	routes []*Route
	ee     []error
}

// Append route.
func (r *Routes) Append(route *Route, err error) {
	if err != nil {
		r.ee = append(r.ee, err)
		return
	}
	if route == nil {
		r.ee = append(r.ee, errors.New("route is nil"))
		return
	}
	r.routes = append(r.routes, route)
}

// Result of the route aggregation.
func (r *Routes) Result() ([]*Route, error) {
	return r.routes, patronerrors.Aggregate(r.ee...)
}
