// Package v2 provides a ready to use HTTP component.
package v2

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/julienschmidt/httprouter"
)

const (
	httpPort            = 50000
	httpReadTimeout     = 30 * time.Second
	httpWriteTimeout    = 60 * time.Second
	httpIdleTimeout     = 240 * time.Second
	shutdownGracePeriod = 5 * time.Second
	deflateLevel        = 6
)

// Component implementation of HTTP.
type Component struct {
	aliveCheck          AliveCheckFunc
	readyCheck          ReadyCheckFunc
	httpPort            int
	httpReadTimeout     time.Duration
	httpWriteTimeout    time.Duration
	deflateLevel        int
	uncompressedPaths   []string
	shutdownGracePeriod time.Duration
	sync.Mutex
	certFile string
	keyFile  string
}

func New(oo ...OptionFunc) (*Component, error) {
	cmp := &Component{
		aliveCheck:          defaultAliveCheck,
		readyCheck:          defaultReadyCheck,
		httpPort:            httpPort,
		httpReadTimeout:     httpReadTimeout,
		httpWriteTimeout:    httpWriteTimeout,
		deflateLevel:        deflateLevel,
		uncompressedPaths:   []string{"/metrics", "/alive", "/ready"},
		shutdownGracePeriod: shutdownGracePeriod,
		certFile:            "",
		keyFile:             "",
	}

	for _, option := range oo {
		err := option(cmp)
		if err != nil {
			return nil, err
		}
	}

	return cmp, nil
}

// Run starts the HTTP server.
func (c *Component) Run(ctx context.Context) error {
	c.Lock()
	log.Debug("applying tracing to routes")
	chFail := make(chan error)
	srv := c.createHTTPServer()
	go c.listenAndServe(srv, chFail)
	c.Unlock()

	select {
	case <-ctx.Done():
		log.Info("shutting down HTTP component")
		tctx, cancel := context.WithTimeout(context.Background(), c.shutdownGracePeriod)
		defer cancel()
		return srv.Shutdown(tctx)
	case err := <-chFail:
		return err
	}
}

func (c *Component) listenAndServe(srv *http.Server, ch chan<- error) {
	if c.certFile != "" && c.keyFile != "" {
		log.Debugf("HTTPS component listening on port %d", c.httpPort)
		ch <- srv.ListenAndServeTLS(c.certFile, c.keyFile)
	}

	log.Debugf("HTTP component listening on port %d", c.httpPort)
	ch <- srv.ListenAndServe()
}

func (c *Component) createHTTPServer() *http.Server {
	log.Debugf("adding %d routes", len(c.routes))
	router := httprouter.New()
	for _, route := range c.routes {
		if len(route.middlewares) > 0 {
			h := MiddlewareChain(route.handler, route.middlewares...)
			router.Handler(route.method, route.path, h)
		} else {
			router.HandlerFunc(route.method, route.path, route.handler)
		}

		log.Debugf("added route %s %s", route.method, route.path)
	}
	// Add first the recovery middleware to ensure that no panic occur.
	routerAfterMiddleware := MiddlewareChain(router, NewRecoveryMiddleware())
	c.middlewares = append(c.middlewares, NewCompressionMiddleware(c.deflateLevel, c.uncompressedPaths...))
	routerAfterMiddleware = MiddlewareChain(routerAfterMiddleware, c.middlewares...)

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", c.httpPort),
		ReadTimeout:  c.httpReadTimeout,
		WriteTimeout: c.httpWriteTimeout,
		IdleTimeout:  httpIdleTimeout,
		Handler:      routerAfterMiddleware,
	}
}

// Builder gathers all required and optional properties, in order
// to construct an HTTP component.
type Builder struct {
	ac                  AliveCheckFunc
	rc                  ReadyCheckFunc
	httpPort            int
	httpReadTimeout     time.Duration
	httpWriteTimeout    time.Duration
	deflateLevel        int
	uncompressedPaths   []string
	shutdownGracePeriod time.Duration
	routesBuilder       *RoutesBuilder
	middlewares         []MiddlewareFunc
	certFile            string
	keyFile             string
	errors              []error
}

// NewBuilder initiates the HTTP component builder chain.
// The builder instantiates the component using default values for
// HTTP Port, Alive/Ready check functions and Read/Write timeouts.
func NewBuilder() *Builder {
	var errs []error
	return &Builder{
		ac:                  DefaultAliveCheck,
		rc:                  DefaultReadyCheck,
		httpPort:            httpPort,
		httpReadTimeout:     httpReadTimeout,
		httpWriteTimeout:    httpWriteTimeout,
		deflateLevel:        deflateLevel,
		uncompressedPaths:   []string{"/metrics", "/alive", "/ready"},
		shutdownGracePeriod: shutdownGracePeriod,
		routesBuilder:       NewRoutesBuilder(),
		errors:              errs,
	}
}

// Create constructs the HTTP component by applying the gathered properties.
func (cb *Builder) Create() (*Component, error) {
	if len(cb.errors) > 0 {
		return nil, patronErrors.Aggregate(cb.errors...)
	}

	for _, rb := range profilingRoutes() {
		cb.routesBuilder.Append(rb)
	}

	routes, err := cb.routesBuilder.Append(aliveCheckRoute(cb.ac)).Append(readyCheckRoute(cb.rc)).
		Append(metricRoute()).Build()
	if err != nil {
		return nil, err
	}

	return &Component{
		aliveCheck:          cb.ac,
		readyCheck:          cb.rc,
		httpPort:            cb.httpPort,
		httpReadTimeout:     cb.httpReadTimeout,
		httpWriteTimeout:    cb.httpWriteTimeout,
		deflateLevel:        cb.deflateLevel,
		uncompressedPaths:   cb.uncompressedPaths,
		shutdownGracePeriod: cb.shutdownGracePeriod,
		routes:              routes,
		middlewares:         cb.middlewares,
		certFile:            cb.certFile,
		keyFile:             cb.keyFile,
	}, nil
}
