package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/beatlabs/patron/log"
	"github.com/julienschmidt/httprouter"
)

const (
	httpPort         = 50000
	httpReadTimeout  = 5 * time.Second
	httpWriteTimeout = 10 * time.Second
	httpIdleTimeout  = 120 * time.Second
)

var (
	// DefaultAliveCheck return always live.
	DefaultAliveCheck = func() AliveStatus { return Alive }
	// DefaultReadyCheck return always ready.
	DefaultReadyCheck = func() ReadyStatus { return Ready }
)

// Component implementation of HTTP.
type Component struct {
	ac               AliveCheckFunc
	rc               ReadyCheckFunc
	httpPort         int
	httpReadTimeout  time.Duration
	httpWriteTimeout time.Duration
	info             map[string]interface{}
	sync.Mutex
	routes      []Route
	middlewares []MiddlewareFunc
	certFile    string
	keyFile     string
}

// New returns a new component.
func New(oo ...OptionFunc) (*Component, error) {
	c := Component{
		ac:               DefaultAliveCheck,
		rc:               DefaultReadyCheck,
		httpPort:         httpPort,
		httpReadTimeout:  httpReadTimeout,
		httpWriteTimeout: httpWriteTimeout,
		routes:           []Route{},
		middlewares:      []MiddlewareFunc{},
		info:             make(map[string]interface{}),
	}

	for _, o := range oo {
		err := o(&c)
		if err != nil {
			return nil, err
		}
	}

	c.routes = append(c.routes, aliveCheckRoute(c.ac))
	c.routes = append(c.routes, readyCheckRoute(c.rc))
	c.routes = append(c.routes, profilingRoutes()...)
	c.routes = append(c.routes, metricRoute())

	return &c, nil
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
		log.Info("shutting down component")
		return srv.Shutdown(ctx)
	case err := <-chFail:
		return err
	}
}

func (c *Component) listenAndServe(srv *http.Server, ch chan<- error) {
	if c.certFile != "" && c.keyFile != "" {
		log.Infof("HTTPS component listening on port %d", c.httpPort)
		ch <- srv.ListenAndServeTLS(c.certFile, c.keyFile)
	}

	log.Infof("HTTP component listening on port %d", c.httpPort)
	ch <- srv.ListenAndServe()
}

func (c *Component) createHTTPServer() *http.Server {
	log.Debugf("adding %d routes", len(c.routes))
	router := httprouter.New()
	for _, route := range c.routes {
		if len(route.Middlewares) > 0 {
			h := MiddlewareChain(route.Handler, route.Middlewares...)
			router.Handler(route.Method, route.Pattern, h)
		} else {
			router.HandlerFunc(route.Method, route.Pattern, route.Handler)
		}

		log.Debugf("added route %s %s", route.Method, route.Pattern)
	}
	// Add first the recovery middleware to ensure that no panic occur.
	routerAfterMiddleware := MiddlewareChain(router, NewRecoveryMiddleware())
	routerAfterMiddleware = MiddlewareChain(routerAfterMiddleware, c.middlewares...)

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", c.httpPort),
		ReadTimeout:  c.httpReadTimeout,
		WriteTimeout: c.httpWriteTimeout,
		IdleTimeout:  httpIdleTimeout,
		Handler:      routerAfterMiddleware,
	}
}
