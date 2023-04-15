package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/beatlabs/patron/component/http/middleware"
	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/exp/slog"
)

const (
	defaultPort                = 50000
	defaultReadTimeout         = 30 * time.Second
	defaultWriteTimeout        = 60 * time.Second
	defaultIdleTimeout         = 240 * time.Second
	defaultShutdownGracePeriod = 5 * time.Second
	defaultDeflateLevel        = 6
)

var (
	// DefaultAliveCheck return always live.
	//
	// Deprecated: Please use the new v2 package.
	// This package is frozen and no new functionality will be added.
	DefaultAliveCheck = func() AliveStatus { return Alive }
	// DefaultReadyCheck return always ready.
	//
	// Deprecated: Please use the new v2 package.
	// This package is frozen and no new functionality will be added.
	DefaultReadyCheck = func() ReadyStatus { return Ready }
)

// Component implementation of HTTP.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
type Component struct {
	ac                  AliveCheckFunc
	rc                  ReadyCheckFunc
	httpPort            int
	httpReadTimeout     time.Duration
	httpWriteTimeout    time.Duration
	deflateLevel        int
	uncompressedPaths   []string
	shutdownGracePeriod time.Duration
	sync.Mutex
	routes      []Route
	middlewares []middleware.Func
	certFile    string
	keyFile     string
}

// Run starts the HTTP server.
func (c *Component) Run(ctx context.Context) error {
	c.Lock()
	slog.Debug("applying tracing to routes")
	chFail := make(chan error)
	srv := c.createHTTPServer()
	go c.listenAndServe(srv, chFail)
	c.Unlock()

	select {
	case <-ctx.Done():
		slog.Info("shutting down HTTP component")
		tctx, cancel := context.WithTimeout(context.Background(), c.shutdownGracePeriod)
		defer cancel()
		return srv.Shutdown(tctx)
	case err := <-chFail:
		return err
	}
}

func (c *Component) listenAndServe(srv *http.Server, ch chan<- error) {
	if c.certFile != "" && c.keyFile != "" {
		slog.Debug("HTTPS component listening", slog.Int("port", c.httpPort))
		ch <- srv.ListenAndServeTLS(c.certFile, c.keyFile)
	}

	slog.Debug("HTTP component listening", slog.Int("port", c.httpPort))
	ch <- srv.ListenAndServe()
}

func (c *Component) createHTTPServer() *http.Server {
	slog.Debug("adding routes", slog.Int("routes", len(c.routes)))
	router := httprouter.New()
	for _, route := range c.routes {
		if len(route.middlewares) > 0 {
			h := middleware.Chain(route.handler, route.middlewares...)
			router.Handler(route.method, route.path, h)
		} else {
			router.HandlerFunc(route.method, route.path, route.handler)
		}

		slog.Debug("added route", slog.String("method", route.method), slog.String("path", route.path))
	}
	// Add first the recovery middleware to ensure that no panic occur.
	routerAfterMiddleware := middleware.Chain(router, middleware.NewRecovery())
	compressionMiddleware, _ := middleware.NewCompression(c.deflateLevel, c.uncompressedPaths...)
	c.middlewares = append(c.middlewares, compressionMiddleware)
	routerAfterMiddleware = middleware.Chain(routerAfterMiddleware, c.middlewares...)

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", c.httpPort),
		ReadTimeout:  c.httpReadTimeout,
		WriteTimeout: c.httpWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
		Handler:      routerAfterMiddleware,
	}
}

func port() (int, error) {
	port, ok := os.LookupEnv("PATRON_HTTP_DEFAULT_PORT")
	if !ok {
		slog.Debug("using default port", slog.Int("port", defaultPort))
		return defaultPort, nil
	}
	portVal, err := strconv.ParseInt(port, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("env var for HTTP default port is not valid: %w", err)
	}
	slog.Debug("using port", slog.Int("port", int(portVal)))
	return int(portVal), nil
}

func readTimeout() (time.Duration, error) {
	httpTimeout, ok := os.LookupEnv("PATRON_HTTP_READ_TIMEOUT")
	if !ok {
		slog.Debug("using default read timeout", slog.Duration("timeout", defaultReadTimeout))
		return defaultReadTimeout, nil
	}
	timeout, err := time.ParseDuration(httpTimeout)
	if err != nil {
		return 0, fmt.Errorf("env var for HTTP read timeout is not valid: %w", err)
	}
	slog.Debug("using read timeout", slog.Duration("timeout", timeout))
	return timeout, nil
}

func writeTimeout() (time.Duration, error) {
	httpTimeout, ok := os.LookupEnv("PATRON_HTTP_WRITE_TIMEOUT")
	if !ok {
		slog.Debug("using default write timeout", slog.Duration("timeout", defaultWriteTimeout))
		return defaultWriteTimeout, nil
	}
	timeout, err := time.ParseDuration(httpTimeout)
	if err != nil {
		return 0, fmt.Errorf("env var for HTTP write timeout is not valid: %w", err)
	}
	slog.Debug("using write timeout", slog.Duration("timeout", timeout))
	return timeout, nil
}

func deflateLevel() (int, error) {
	deflateLevel, ok := os.LookupEnv("PATRON_COMPRESSION_DEFLATE_LEVEL")
	if !ok {
		slog.Debug("using default deflate level", slog.Int("level", defaultDeflateLevel))
		return defaultDeflateLevel, nil
	}
	deflateLevelInt, err := strconv.Atoi(deflateLevel)
	if err != nil {
		return 0, fmt.Errorf("env var for HTTP deflate level is not valid: %w", err)
	}
	slog.Debug("using deflate level", slog.Int("level", deflateLevelInt))
	return deflateLevelInt, nil
}

// Builder gathers all required and optional properties, in order
// to construct an HTTP component.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
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
	middlewares         []middleware.Func
	certFile            string
	keyFile             string
	errors              []error
}

// NewBuilder initiates the HTTP component builder chain.
// The builder instantiates the component using default values for
// HTTP Port, Alive/Ready check functions and Read/Write timeouts.
//
// Deprecated: Please use the new v2 package.
// This package is frozen and no new functionality will be added.
func NewBuilder() *Builder {
	var ee []error

	port, err := port()
	if err != nil {
		ee = append(ee, err)
	}

	readTimeout, err := readTimeout()
	if err != nil {
		ee = append(ee, err)
	}

	writeTimeout, err := writeTimeout()
	if err != nil {
		ee = append(ee, err)
	}

	deflateLevel, err := deflateLevel()
	if err != nil {
		ee = append(ee, err)
	}

	return &Builder{
		ac:                  DefaultAliveCheck,
		rc:                  DefaultReadyCheck,
		httpPort:            port,
		httpReadTimeout:     readTimeout,
		httpWriteTimeout:    writeTimeout,
		deflateLevel:        deflateLevel,
		uncompressedPaths:   []string{MetricsPath, AlivePath, ReadyPath},
		shutdownGracePeriod: defaultShutdownGracePeriod,
		routesBuilder:       NewRoutesBuilder(),
		errors:              ee,
	}
}

// WithSSL sets the filenames for the Certificate and Keyfile, in order to enable SSL.
func (cb *Builder) WithSSL(c, k string) *Builder {
	if c == "" || k == "" {
		cb.errors = append(cb.errors, errors.New("invalid cert or key provided"))
	} else {
		slog.Debug("setting cert file and key")
		cb.certFile = c
		cb.keyFile = k
	}

	return cb
}

// WithRoutesBuilder adds routes builder to the HTTP component.
func (cb *Builder) WithRoutesBuilder(rb *RoutesBuilder) *Builder {
	if rb == nil {
		cb.errors = append(cb.errors, errors.New("route builder is nil"))
	} else {
		slog.Debug("setting route builder")
		cb.routesBuilder = rb
	}
	return cb
}

// WithMiddlewares adds middlewares to the HTTP component.
func (cb *Builder) WithMiddlewares(mm ...middleware.Func) *Builder {
	if len(mm) == 0 {
		cb.errors = append(cb.errors, errors.New("empty list of middlewares provided"))
	} else {
		slog.Debug("setting middlewares")
		cb.middlewares = append(cb.middlewares, mm...)
	}

	return cb
}

// WithReadTimeout sets the Read WithTimeout for the HTTP component.
func (cb *Builder) WithReadTimeout(rt time.Duration) *Builder {
	if rt <= 0*time.Second {
		cb.errors = append(cb.errors, errors.New("negative or zero read timeout provided"))
	} else {
		slog.Debug("setting read timeout")
		cb.httpReadTimeout = rt
	}

	return cb
}

// WithWriteTimeout sets write timeout for the HTTP component.
func (cb *Builder) WithWriteTimeout(wt time.Duration) *Builder {
	if wt <= 0*time.Second {
		cb.errors = append(cb.errors, errors.New("negative or zero write timeout provided"))
	} else {
		slog.Debug("setting write timeout")
		cb.httpWriteTimeout = wt
	}

	return cb
}

// WithDeflateLevel sets the level of compression for Deflate; based on https://golang.org/pkg/compress/flate/
// Levels range from 1 (BestSpeed) to 9 (BestCompression); higher levels typically run slower but compress more.
// Level 0 (NoCompression) does not attempt any compression; it only adds the necessary DEFLATE framing.
// Level -1 (DefaultCompression) uses the default compression level.
// Level -2 (HuffmanOnly) will use Huffman compression only, giving a very fast compression for all types of input, but sacrificing considerable compression efficiency.
func (cb *Builder) WithDeflateLevel(level int) *Builder {
	if level < -2 || level > 9 {
		cb.errors = append(cb.errors, errors.New("provided deflate level value not in the [-2, 9] range"))
	} else {
		cb.deflateLevel = level
	}
	return cb
}

// WithUncompressedPaths specifies which routes should be excluded from compression
// Any trailing slashes are trimmed, so we match both /metrics/ and /metrics?seconds=30.
func (cb *Builder) WithUncompressedPaths(r ...string) *Builder {
	res := make([]string, 0, len(r))
	for _, e := range r {
		for len(e) > 1 && e[len(e)-1] == '/' {
			e = e[0 : len(e)-1]
		}
		res = append(res, e)
	}
	cb.uncompressedPaths = append(cb.uncompressedPaths, res...)

	return cb
}

// WithShutdownGracePeriod sets the Shutdown Grace Period for the HTTP component.
func (cb *Builder) WithShutdownGracePeriod(gp time.Duration) *Builder {
	if gp <= 0*time.Second {
		cb.errors = append(cb.errors, errors.New("negative or zero shutdown grace period provided"))
	} else {
		slog.Debug("setting shutdown grace period")
		cb.shutdownGracePeriod = gp
	}

	return cb
}

// WithPort sets the port used by the HTTP component.
func (cb *Builder) WithPort(p int) *Builder {
	if p <= 0 || p > 65535 {
		cb.errors = append(cb.errors, errors.New("invalid HTTP Port provided"))
	} else {
		slog.Debug("setting port")
		cb.httpPort = p
	}

	return cb
}

// WithAliveCheckFunc sets the AliveCheckFunc used by the HTTP component.
func (cb *Builder) WithAliveCheckFunc(acf AliveCheckFunc) *Builder {
	if acf == nil {
		cb.errors = append(cb.errors, errors.New("nil AliveCheckFunc was provided"))
	} else {
		slog.Debug("setting aliveness check")
		cb.ac = acf
	}

	return cb
}

// WithReadyCheckFunc sets the ReadyCheckFunc used by the HTTP component.
func (cb *Builder) WithReadyCheckFunc(rcf ReadyCheckFunc) *Builder {
	if rcf == nil {
		cb.errors = append(cb.errors, errors.New("nil ReadyCheckFunc provided"))
	} else {
		slog.Debug("setting readiness check")
		cb.rc = rcf
	}

	return cb
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
		ac:                  cb.ac,
		rc:                  cb.rc,
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
