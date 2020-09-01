package patron

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/beatlabs/patron/component/http"
	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/log/zerolog"
	"github.com/beatlabs/patron/trace"
	jaeger "github.com/uber/jaeger-client-go"
)

var logSetupOnce sync.Once

const (
	srv  = "srv"
	ver  = "ver"
	host = "host"
)

// SetupLogging sets up the default metrics logging.
func SetupLogging(name, version string) error {
	return setupLogging(name, version, map[string]interface{}{})
}

// SetupLoggingWithFields sets up the default metrics logging with given fields.
func SetupLoggingWithFields(name, version string, fields map[string]interface{}) error {
	return setupLogging(name, version, fields)
}

func setupLogging(name, version string, fields map[string]interface{}) error {
	lvl, ok := os.LookupEnv("PATRON_LOG_LEVEL")
	if !ok {
		lvl = string(log.InfoLevel)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	f := map[string]interface{}{
		srv:  name,
		ver:  version,
		host: hostname,
	}

	for k, v := range fields {
		if k == srv || k == ver || k == host {
			// don't override
			continue
		}
		f[k] = v
	}

	logSetupOnce.Do(func() {
		err = log.Setup(zerolog.Create(log.Level(lvl)), f)
	})

	return err
}

// Component interface for implementing service components.
type Component interface {
	Run(ctx context.Context) error
}

// service is responsible for managing and setting up everything.
// The service will start by default a HTTP component in order to host management endpoint.
type service struct {
	cps           []Component
	routesBuilder *http.RoutesBuilder
	middlewares   []http.MiddlewareFunc
	acf           http.AliveCheckFunc
	rcf           http.ReadyCheckFunc
	termSig       chan os.Signal
	sighupHandler func()
}

func (s *service) setupOSSignal() {
	signal.Notify(s.termSig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
}

func (s *service) run(ctx context.Context) error {
	defer func() {
		err := trace.Close()
		if err != nil {
			log.Errorf("failed to close trace %v", err)
		}
	}()
	cctx, cnl := context.WithCancel(ctx)
	chErr := make(chan error, len(s.cps))
	wg := sync.WaitGroup{}
	wg.Add(len(s.cps))
	for _, cp := range s.cps {
		go func(c Component) {
			defer wg.Done()
			chErr <- c.Run(cctx)
		}(cp)
	}

	ee := make([]error, 0, len(s.cps))
	ee = append(ee, s.waitTermination(chErr))
	cnl()

	wg.Wait()
	close(chErr)

	for err := range chErr {
		ee = append(ee, err)
	}
	return patronErrors.Aggregate(ee...)
}

func (s *service) setupDefaultTracing(name, version string) error {
	var err error

	host, ok := os.LookupEnv("PATRON_JAEGER_AGENT_HOST")
	if !ok {
		host = "0.0.0.0"
	}
	port, ok := os.LookupEnv("PATRON_JAEGER_AGENT_PORT")
	if !ok {
		port = "6831"
	}
	agent := host + ":" + port
	tp, ok := os.LookupEnv("PATRON_JAEGER_SAMPLER_TYPE")
	if !ok {
		tp = jaeger.SamplerTypeProbabilistic
	}
	prmVal := 0.0
	prm := "0.0"

	if prm, ok := os.LookupEnv("PATRON_JAEGER_SAMPLER_PARAM"); ok {
		prmVal, err = strconv.ParseFloat(prm, 64)
		if err != nil {
			return fmt.Errorf("env var for jaeger sampler param is not valid: %w", err)
		}
	}

	log.Infof("setting up default tracing %s, %s with param %s", agent, tp, prm)
	return trace.Setup(name, version, agent, tp, prmVal)
}

func (s *service) createHTTPComponent() (Component, error) {
	var err error
	portVal := int64(50000)
	port, ok := os.LookupEnv("PATRON_HTTP_DEFAULT_PORT")
	if ok {
		portVal, err = strconv.ParseInt(port, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("env var for HTTP default port is not valid: %w", err)
		}
	}
	port = strconv.FormatInt(portVal, 10)
	log.Infof("creating default HTTP component at port %s", port)

	b := http.NewBuilder().WithPort(int(portVal))

	httpReadTimeout, ok := os.LookupEnv("PATRON_HTTP_READ_TIMEOUT")
	if ok {
		readTimeout, err := time.ParseDuration(httpReadTimeout)
		if err != nil {
			return nil, fmt.Errorf("env var for HTTP read timeout is not valid: %w", err)
		}
		b.WithReadTimeout(readTimeout)
		log.Infof("setting up default HTTP read timeout %s", httpReadTimeout)
	}

	httpWriteTimeout, ok := os.LookupEnv("PATRON_HTTP_WRITE_TIMEOUT")
	if ok {
		writeTimeout, err := time.ParseDuration(httpWriteTimeout)
		if err != nil {
			return nil, fmt.Errorf("env var for HTTP write timeout is not valid: %w", err)
		}
		b.WithWriteTimeout(writeTimeout)
		log.Infof("setting up default HTTP write timeout %s", httpWriteTimeout)
	}

	if s.acf != nil {
		b.WithAliveCheckFunc(s.acf)
	}

	if s.rcf != nil {
		b.WithReadyCheckFunc(s.rcf)
	}

	if s.routesBuilder != nil {
		b.WithRoutesBuilder(s.routesBuilder)
	}

	if s.middlewares != nil && len(s.middlewares) > 0 {
		b.WithMiddlewares(s.middlewares...)
	}

	cp, err := b.Create()
	if err != nil {
		return nil, fmt.Errorf("failed to create default HTTP component: %w", err)
	}

	return cp, nil
}

func (s *service) waitTermination(chErr <-chan error) error {
	for {
		select {
		case sig := <-s.termSig:
			log.Infof("signal %s received", sig.String())
			switch sig {
			case syscall.SIGHUP:
				s.sighupHandler()
			default:
				return nil
			}
		case err := <-chErr:
			log.Info("component error received")
			return err
		}
	}
}

// Builder gathers all required properties to
// construct a Patron service.
type Builder struct {
	errors        []error
	name          string
	version       string
	fields        map[string]interface{}
	cps           []Component
	routesBuilder *http.RoutesBuilder
	middlewares   []http.MiddlewareFunc
	acf           http.AliveCheckFunc
	rcf           http.ReadyCheckFunc
	termSig       chan os.Signal
	sighupHandler func()
}

// New initiates the Service builder chain.
// The builder contains default values for Alive/Ready checks,
// the SIGHUP handler and its version.
func New(name, version string) *Builder {
	var errs []error

	if name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if version == "" {
		version = "dev"
	}

	return &Builder{
		errors:        errs,
		name:          name,
		version:       version,
		acf:           http.DefaultAliveCheck,
		rcf:           http.DefaultReadyCheck,
		termSig:       make(chan os.Signal, 1),
		sighupHandler: func() { log.Info("SIGHUP received: nothing setup") },
	}
}

// WithRoutesBuilder adds routes builder to the default HTTP component.
func (b *Builder) WithRoutesBuilder(rb *http.RoutesBuilder) *Builder {
	if rb == nil {
		b.errors = append(b.errors, errors.New("routes builder is nil"))
	} else {
		log.Info("setting routes builder")
		b.routesBuilder = rb
	}

	return b
}

// WithMiddlewares adds generic middlewares to the default HTTP component.
func (b *Builder) WithMiddlewares(mm ...http.MiddlewareFunc) *Builder {
	if len(mm) == 0 {
		b.errors = append(b.errors, errors.New("provided middlewares slice was empty"))
	} else {
		log.Info("setting middlewares")
		b.middlewares = append(b.middlewares, mm...)
	}

	return b
}

// WithAliveCheck overrides the default liveness check of the default HTTP component.
func (b *Builder) WithAliveCheck(acf http.AliveCheckFunc) *Builder {
	if acf == nil {
		b.errors = append(b.errors, errors.New("alive check func provided was nil"))
	} else {
		log.Info("setting alive check func")
		b.acf = acf
	}

	return b
}

// WithReadyCheck overrides the default readiness check of the default HTTP component.
func (b *Builder) WithReadyCheck(rcf http.ReadyCheckFunc) *Builder {
	if rcf == nil {
		b.errors = append(b.errors, errors.New("ready check func provided was nil"))
	} else {
		log.Info("setting ready check func")
		b.rcf = rcf
	}

	return b
}

// WithComponents adds custom components to the Patron service.
func (b *Builder) WithComponents(cc ...Component) *Builder {
	if len(cc) == 0 {
		b.errors = append(b.errors, errors.New("provided components slice was empty"))
	} else {
		log.Info("setting components")
		b.cps = append(b.cps, cc...)
	}

	return b
}

// WithSIGHUP adds a custom handler for when the service receives a SIGHUP.
func (b *Builder) WithSIGHUP(handler func()) *Builder {
	if handler == nil {
		b.errors = append(b.errors, errors.New("provided SIGHUP handler was nil"))
	} else {
		log.Info("setting SIGHUP handler func")
		b.sighupHandler = handler
	}

	return b
}

// WithLogFields sets key/value structured log fields to be passed to the logger.
func (b *Builder) WithLogFields(fields map[string]interface{}) *Builder {
	b.fields = fields
	return b
}

// Build constructs the Patron service by applying the gathered properties.
func (b *Builder) build() (*service, error) {
	if len(b.errors) > 0 {
		return nil, patronErrors.Aggregate(b.errors...)
	}

	s := service{
		cps:           b.cps,
		routesBuilder: b.routesBuilder,
		middlewares:   b.middlewares,
		acf:           b.acf,
		rcf:           b.rcf,
		termSig:       b.termSig,
		sighupHandler: b.sighupHandler,
	}

	err := SetupLoggingWithFields(b.name, b.version, b.fields)
	if err != nil {
		return nil, err
	}

	err = s.setupDefaultTracing(b.name, b.version)
	if err != nil {
		return nil, err
	}

	httpCp, err := s.createHTTPComponent()
	if err != nil {
		return nil, err
	}

	s.cps = append(s.cps, httpCp)
	s.setupOSSignal()
	return &s, nil
}

// Run starts up all service components and monitors for errors.
// If a component returns a error the service is responsible for shutting down
// all components and terminate itself.
func (b *Builder) Run(ctx context.Context) error {
	s, err := b.build()
	if err != nil {
		return err
	}

	return s.run(ctx)
}
