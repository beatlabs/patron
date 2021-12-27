package v2

import (
	"errors"
	"net/http"

	patronhttp "github.com/beatlabs/patron/component/http"
)

// RouteOptionFunc definition for configuring the route in a functional way.
type RouteOptionFunc func(route *Route) error

// Route definition of a HTTP route.
type Route struct {
	method      string
	path        string
	handler     http.HandlerFunc
	middlewares []patronhttp.MiddlewareFunc
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

func (r Route) Middlewares() []patronhttp.MiddlewareFunc {
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
		method:      method,
		path:        path,
		handler:     handler,
		middlewares: []patronhttp.MiddlewareFunc{patronhttp.NewRecoveryMiddleware()},
	}

	for _, option := range oo {
		err := option(route)
		if err != nil {
			return nil, err
		}
	}

	return route, nil
}
