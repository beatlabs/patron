// Package v2 provides a ready to use HTTP component.
package v2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/beatlabs/patron/log"
)

const (
	port                = 50000
	readTimeout         = 30 * time.Second
	writeTimeout        = 60 * time.Second
	idleTimeout         = 240 * time.Second
	handlerTimeout      = 59 * time.Second // should be smaller than write timeout
	shutdownGracePeriod = 5 * time.Second
)

// Component implementation of an HTTP router.
type Component struct {
	port                int
	readTimeout         time.Duration
	writeTimeout        time.Duration
	shutdownGracePeriod time.Duration
	handlerTimeout      time.Duration
	handler             http.Handler
	mu                  sync.Mutex
	certFile            string
	keyFile             string
}

// New creates an HTTP component configurable by functional options.
func New(handler http.Handler, oo ...OptionFunc) (*Component, error) {
	if handler == nil {
		return nil, errors.New("handler is nil")
	}

	cmp := &Component{
		port:                port,
		readTimeout:         readTimeout,
		writeTimeout:        writeTimeout,
		shutdownGracePeriod: shutdownGracePeriod,
		handlerTimeout:      handlerTimeout,
		handler:             handler,
	}

	for _, option := range oo {
		err := option(cmp)
		if err != nil {
			return nil, err
		}
	}

	return cmp, nil
}

// Run starts the HTTP server and returns only if listening and/or serving failed, or if the context was canceled.
func (c *Component) Run(ctx context.Context) error {
	c.mu.Lock()
	chFail := make(chan error)
	srv := c.createHTTPServer()
	go c.listenAndServe(srv, chFail)
	c.mu.Unlock()

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
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", c.port),
		ReadTimeout:  c.readTimeout,
		WriteTimeout: c.writeTimeout,
		IdleTimeout:  idleTimeout,
		Handler:      http.TimeoutHandler(c.handler, c.handlerTimeout, ""),
	}
}

func (c *Component) listenAndServe(srv *http.Server, ch chan<- error) {
	if c.certFile != "" && c.keyFile != "" {
		log.Debugf("HTTPS component listening on port %d", c.port)
		ch <- srv.ListenAndServeTLS(c.certFile, c.keyFile)
		return
	}

	log.Debugf("HTTP component listening on port %d", c.port)
	ch <- srv.ListenAndServe()
}
