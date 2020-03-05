package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	patronErrors "github.com/beatlabs/patron/errors"
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
	sync.Mutex
	routes      []Route
	middlewares []MiddlewareFunc
	certFile    string
	keyFile     string
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

const fieldSetMsg = "Setting property '%v' for '%v'"

// Builder gathers all required and optional properties, in order
// to construct an HTTP component.
type Builder struct {
	ac               AliveCheckFunc
	rc               ReadyCheckFunc
	httpPort         int
	httpReadTimeout  time.Duration
	httpWriteTimeout time.Duration
	routes           []Route
	middlewares      []MiddlewareFunc
	certFile         string
	keyFile          string
	errors           []error
}

// NewBuilder initiates the HTTP component builder chain.
// The builder instantiates the component using default values for
// HTTP Port, Alive/Ready check functions and Read/Write timeouts.
func NewBuilder() *Builder {
	var errs []error
	return &Builder{
		ac:               DefaultAliveCheck,
		rc:               DefaultReadyCheck,
		httpPort:         httpPort,
		httpReadTimeout:  httpReadTimeout,
		httpWriteTimeout: httpWriteTimeout,
		errors:           errs,
	}
}

// WithSSL sets the filenames for the Certificate and Keyfile, in order to enable SSL.
func (cb *Builder) WithSSL(c, k string) *Builder {
	if c == "" || k == "" {
		cb.errors = append(cb.errors, errors.New("Invalid cert or key provided"))
	} else {
		log.Info(fieldSetMsg, "Cert, Key", c+","+k)
		cb.certFile = c
		cb.keyFile = k
	}

	return cb
}

// WithRoutes adds routes to the HTTP component.
func (cb *Builder) WithRoutes(rr []Route) *Builder {
	if len(rr) == 0 {
		cb.errors = append(cb.errors, errors.New("Empty Routes slice provided"))
	} else {
		log.Info(fieldSetMsg, "Routes", rr)
		cb.routes = append(cb.routes, rr...)
	}

	return cb
}

// WithMiddlewares adds middlewares to the HTTP component.
func (cb *Builder) WithMiddlewares(mm ...MiddlewareFunc) *Builder {
	if len(mm) == 0 {
		cb.errors = append(cb.errors, errors.New("Empty list of middlewares provided"))
	} else {
		log.Info(fieldSetMsg, "Middlewares", mm)
		cb.middlewares = append(cb.middlewares, mm...)
	}

	return cb
}

// WithReadTimeout sets the Read Timeout for the HTTP component.
func (cb *Builder) WithReadTimeout(rt time.Duration) *Builder {
	if rt <= 0*time.Second {
		cb.errors = append(cb.errors, errors.New("Negative or zero read timeout provided"))
	} else {
		log.Infof(fieldSetMsg, "Read Timeout", rt)
		cb.httpReadTimeout = rt
	}

	return cb
}

// WithWriteTimeout sets the Write Timeout for the HTTP component.
func (cb *Builder) WithWriteTimeout(wt time.Duration) *Builder {
	if wt <= 0*time.Second {
		cb.errors = append(cb.errors, errors.New("Negative or zero write timeout provided"))
	} else {
		log.Infof(fieldSetMsg, "Write Timeout", wt)
		cb.httpWriteTimeout = wt
	}

	return cb
}

// WithPort sets the port used by the HTTP component.
func (cb *Builder) WithPort(p int) *Builder {
	if p <= 0 || p > 65535 {
		cb.errors = append(cb.errors, errors.New("Invalid HTTP Port provided"))
	} else {
		log.Infof(fieldSetMsg, "Port", p)
		cb.httpPort = p
	}

	return cb
}

// WithAliveCheckFunc sets the AliveCheckFunc used by the HTTP component.
func (cb *Builder) WithAliveCheckFunc(acf AliveCheckFunc) *Builder {
	if acf == nil {
		cb.errors = append(cb.errors, errors.New("Nil AliveCheckFunc was provided"))
	} else {
		log.Infof(fieldSetMsg, "AliveCheckFunc", acf)
		cb.ac = acf
	}

	return cb
}

// WithReadyCheckFunc sets the ReadyCheckFunc used by the HTTP component.
func (cb *Builder) WithReadyCheckFunc(rcf ReadyCheckFunc) *Builder {
	if rcf == nil {
		cb.errors = append(cb.errors, errors.New("Nil ReadyCheckFunc provided"))
	} else {
		log.Infof(fieldSetMsg, "ReadyCheckFunc", rcf)
		cb.rc = rcf
	}

	return cb
}

// Create constructs the HTTP component by applying the gathered properties.
func (cb *Builder) Create() (*Component, error) {
	if len(cb.errors) > 0 {
		return nil, patronErrors.Aggregate(cb.errors...)
	}

	c := &Component{
		ac:               cb.ac,
		rc:               cb.rc,
		httpPort:         cb.httpPort,
		httpReadTimeout:  cb.httpReadTimeout,
		httpWriteTimeout: cb.httpWriteTimeout,
		routes:           cb.routes,
		middlewares:      cb.middlewares,
		certFile:         cb.certFile,
		keyFile:          cb.keyFile,
	}

	c.routes = append(c.routes, aliveCheckRoute(c.ac))
	c.routes = append(c.routes, readyCheckRoute(c.rc))
	c.routes = append(c.routes, profilingRoutes()...)
	c.routes = append(c.routes, metricRoute())

	return c, nil
}
