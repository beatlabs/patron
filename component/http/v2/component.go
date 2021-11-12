// Package v2 provides a ready to use HTTP component.
package v2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/log"
	"github.com/gorilla/mux"
)

const (
	port                = 50000
	readTimeout         = 30 * time.Second
	writeTimeout        = 60 * time.Second
	idleTimeout         = 240 * time.Second
	shutdownGracePeriod = 5 * time.Second
	deflateLevel        = 6
)

// Component implementation of HTTP.
type Component struct {
	aliveCheck          patronhttp.AliveCheckFunc
	readyCheck          patronhttp.ReadyCheckFunc
	port                int
	readTimeout         time.Duration
	writeTimeout        time.Duration
	deflateLevel        int
	uncompressedPaths   []string
	shutdownGracePeriod time.Duration
	mux                 *mux.Router
	sync.Mutex
	certFile string
	keyFile  string
}

func New(mux *mux.Router, oo ...OptionFunc) (*Component, error) {
	if mux == nil {
		return nil, errors.New("handler is nil")
	}

	cmp := &Component{
		aliveCheck:          defaultAliveCheck,
		readyCheck:          defaultReadyCheck,
		port:                port,
		readTimeout:         readTimeout,
		writeTimeout:        writeTimeout,
		deflateLevel:        deflateLevel,
		uncompressedPaths:   []string{patronhttp.MetricsPath, patronhttp.AlivePath, patronhttp.ReadyPath},
		shutdownGracePeriod: shutdownGracePeriod,
		mux:                 mux,
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
	chFail := make(chan error)
	srv := c.createHTTPServer()
	go c.listenAndServe(srv, chFail)
	c.Unlock()

	select {
	case <-ctx.Done():
		log.Info("shutting down HTTP component")
		ctx, cancel := context.WithTimeout(context.Background(), c.shutdownGracePeriod)
		defer cancel()
		return srv.Shutdown(ctx)
	case err := <-chFail:
		return err
	}
}

func (c *Component) createHTTPServer() *http.Server {
	registerAliveCheckHandler(c.mux, c.aliveCheck)
	registerReadyCheckHandler(c.mux, c.readyCheck)
	registerMetricsHandler(c.mux)
	registerPprofHandlers(c.mux)
	// router := httprouter.New()
	// for _, route := range c.routes {
	// 	if len(route.middlewares) > 0 {
	// 		h := MiddlewareChain(route.handler, route.middlewares...)
	// 		router.Handler(route.method, route.path, h)
	// 	} else {
	// 		router.HandlerFunc(route.method, route.path, route.handler)
	// 	}

	// 	log.Debugf("added route %s %s", route.method, route.path)
	// }
	// // Add first the recovery middleware to ensure that no panic occur.
	// routerAfterMiddleware := MiddlewareChain(router, NewRecoveryMiddleware())
	// c.middlewares = append(c.middlewares, NewCompressionMiddleware(c.deflateLevel, c.uncompressedPaths...))
	// routerAfterMiddleware = MiddlewareChain(routerAfterMiddleware, c.middlewares...)

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", c.port),
		ReadTimeout:  c.readTimeout,
		WriteTimeout: c.writeTimeout,
		IdleTimeout:  idleTimeout,
		Handler:      c.mux,
	}
}

func (c *Component) listenAndServe(srv *http.Server, ch chan<- error) {
	if c.certFile != "" && c.keyFile != "" {
		log.Debugf("HTTPS component listening on port %d", c.port)
		ch <- srv.ListenAndServeTLS(c.certFile, c.keyFile)
	}

	log.Debugf("HTTP component listening on port %d", c.port)
	ch <- srv.ListenAndServe()
}
