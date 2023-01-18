// Package v2 provides a ready to use HTTP component.
package v2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/beatlabs/patron/log"
)

const (
	defaultPort                = 50000
	defaultReadTimeout         = 30 * time.Second
	defaultWriteTimeout        = 60 * time.Second
	defaultIdleTimeout         = 240 * time.Second
	defaultHandlerTimeout      = 59 * time.Second // should be smaller than write timeout
	defaultShutdownGracePeriod = 5 * time.Second
)

func port() (int, error) {
	var err error
	portVal := int64(defaultPort)
	port, ok := os.LookupEnv("PATRON_HTTP_DEFAULT_PORT")
	if !ok {
		log.Debugf("using default port %d", defaultPort)
		return defaultPort, nil
	}
	portVal, err = strconv.ParseInt(port, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("env var for HTTP default port is not valid: %w", err)
	}
	log.Debugf("using port %d", portVal)
	return int(portVal), nil
}

func readTimeout() (time.Duration, error) {
	httpTimeout, ok := os.LookupEnv("PATRON_HTTP_READ_TIMEOUT")
	if !ok {
		log.Debugf("using default read timeout %s", defaultReadTimeout)
		return defaultReadTimeout, nil
	}
	timeout, err := time.ParseDuration(httpTimeout)
	if err != nil {
		return 0, fmt.Errorf("env var for HTTP read timeout is not valid: %w", err)
	}
	log.Debugf("using read timeout %s", timeout)
	return timeout, nil
}

func writeTimeout() (time.Duration, error) {
	httpTimeout, ok := os.LookupEnv("PATRON_HTTP_WRITE_TIMEOUT")
	if !ok {
		log.Debugf("using default write timeout %s", defaultWriteTimeout)
		return defaultWriteTimeout, nil
	}
	timeout, err := time.ParseDuration(httpTimeout)
	if err != nil {
		return 0, fmt.Errorf("env var for HTTP write timeout is not valid: %w", err)
	}
	log.Debugf("using write timeout %s", timeout)
	return timeout, nil
}

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

	port, err := port()
	if err != nil {
		return nil, err
	}

	readTimeout, err := readTimeout()
	if err != nil {
		return nil, err
	}

	writeTimeout, err := writeTimeout()
	if err != nil {
		return nil, err
	}

	cmp := &Component{
		port:                port,
		readTimeout:         readTimeout,
		writeTimeout:        writeTimeout,
		shutdownGracePeriod: defaultShutdownGracePeriod,
		handlerTimeout:      defaultHandlerTimeout,
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
		IdleTimeout:  defaultIdleTimeout,
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
