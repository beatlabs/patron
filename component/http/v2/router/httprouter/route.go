package httprouter

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
)

// RouteOptionFunc definition for configuring the route in a functional way.
type RouteOptionFunc func(route *Route) error

// Route definition of a HTTP route.
type Route struct {
	path        string
	method      string
	handler     http.HandlerFunc
	middlewares []MiddlewareFunc
}

// NewRawRoute creates a new raw route with functional configuration.
func NewRawRoute(method, path string, handler http.HandlerFunc, oo ...RouteOptionFunc) (*Route, error) {
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

// NewRoute returns a route that has default middlewares injected.
func NewRoute(method, path string, handler http.HandlerFunc, oo ...RouteOptionFunc) (*Route, error) {
	// TODO: we need something smarter
	// parse a list of HTTP numeric status codes that must be logged
	cfg, _ := os.LookupEnv("PATRON_HTTP_STATUS_ERROR_LOGGING")
	statusCodeLogger, err := newStatusCodeLoggerHandler(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse status codes %q: %w", cfg, err)
	}

	options := make([]RouteOptionFunc, 0, len(oo)+1)
	// prepend standard middlewares
	options = append(options, Middlewares(NewRecoveryMiddleware(), NewLoggingTracingMiddleware(path, statusCodeLogger),
		NewRequestObserverMiddleware(method, path)))
	options = append(options, oo...)

	route, err := NewRawRoute(method, path, handler, oo...)
	if err != nil {
		return nil, err
	}

	return route, nil
}

func NewRecoveryGetRoute(path string, handler http.HandlerFunc) (*Route, error) {
	return NewRawRoute(http.MethodGet, path, handler, Middlewares(NewRecoveryMiddleware()))
}

// NewFileServerRoute returns a route that acts as a file server.
func NewFileServerRoute(path string, assetsDir string, fallbackPath string) (*Route, error) {
	if path == "" {
		return nil, errors.New("path is empty")
	}

	if assetsDir == "" {
		return nil, errors.New("assets path is empty")
	}

	if fallbackPath == "" {
		return nil, errors.New("fallback path is empty")
	}

	_, err := os.Stat(assetsDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("assets directory [%s] doesn't exist", path)
	} else if err != nil {
		return nil, fmt.Errorf("error while checking assets dir: %w", err)
	}

	_, err = os.Stat(fallbackPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("fallback file [%s] doesn't exist", fallbackPath)
	} else if err != nil {
		return nil, fmt.Errorf("error while checking fallback file: %w", err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		paramPath := ""
		for _, param := range httprouter.ParamsFromContext(r.Context()) {
			if param.Key == "path" {
				paramPath = param.Value
				break
			}
		}

		// get the absolute path to prevent directory traversal
		path := assetsDir + paramPath

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

	return NewRawRoute(http.MethodGet, path, handler)
}
