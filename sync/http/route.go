package http

import (
	"net/http"

	"github.com/beatlabs/patron/sync"
	"github.com/beatlabs/patron/sync/http/auth"
)

// Route definition of a HTTP route.
type Route struct {
	Pattern     string
	Method      string
	Handler     http.HandlerFunc
	Trace       bool
	Auth        auth.Authenticator
	Middlewares []MiddlewareFunc
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
		middlewares = append(middlewares, NewTracingMiddleware(p))
	}
	if auth != nil {
		middlewares = append(middlewares, NewAuthMiddleware(auth))
	}
	if len(mm) > 0 {
		middlewares = append(middlewares, mm...)
	}
	return Route{Pattern: p, Method: m, Handler: handler(pr), Trace: trace, Auth: auth, Middlewares: middlewares}
}

// NewRouteRaw creates a new route from a HTTP handler.
func NewRouteRaw(p string, m string, h http.HandlerFunc, trace bool, mm ...MiddlewareFunc) Route {
	var middlewares []MiddlewareFunc
	if trace {
		middlewares = append(middlewares, NewTracingMiddleware(p))
	}
	if len(mm) > 0 {
		middlewares = append(middlewares, mm...)
	}
	return Route{Pattern: p, Method: m, Handler: h, Trace: trace, Middlewares: middlewares}
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
		middlewares = append(middlewares, NewTracingMiddleware(p))
	}
	if auth != nil {
		middlewares = append(middlewares, NewAuthMiddleware(auth))
	}
	if len(mm) > 0 {
		middlewares = append(middlewares, mm...)
	}
	return Route{Pattern: p, Method: m, Handler: h, Trace: trace, Auth: auth, Middlewares: middlewares}
}
