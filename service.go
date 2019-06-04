package patron

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/info"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/log/zerolog"
	"github.com/beatlabs/patron/sync/http"
	"github.com/beatlabs/patron/trace"
	jaeger "github.com/uber/jaeger-client-go"
)

var logSetupOnce sync.Once

// Component interface for implementing service components.
type Component interface {
	Run(ctx context.Context) error
	Info() map[string]interface{}
}

// Service is responsible for managing and setting up everything.
// The service will start by default a HTTP component in order to host management endpoint.
type Service struct {
	cps           []Component
	routes        []http.Route
	middlewares   []http.MiddlewareFunc
	hcf           http.HealthCheckFunc
	termSig       chan os.Signal
	sighupHandler func()
}

// New creates a new named service and allows for customization through functional options.
func New(name, version string, oo ...OptionFunc) (*Service, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}

	if version == "" {
		version = "dev"
	}
	info.UpdateName(name)
	info.UpdateVersion(version)

	s := Service{
		cps:           []Component{},
		hcf:           http.DefaultHealthCheck,
		termSig:       make(chan os.Signal, 1),
		sighupHandler: func() { log.Info("SIGHUP received: nothing setup") },
		middlewares:   []http.MiddlewareFunc{},
	}

	err := Setup(name, version)
	if err != nil {
		return nil, err
	}

	err = s.setupDefaultTracing(name, version)
	if err != nil {
		return nil, err
	}

	for _, o := range oo {
		err = o(&s)
		if err != nil {
			return nil, err
		}
	}

	httpCp, err := s.createHTTPComponent()
	if err != nil {
		return nil, err
	}

	s.cps = append(s.cps, httpCp)
	s.setupInfo()
	s.setupOSSignal()
	return &s, nil
}

func (s *Service) setupOSSignal() {
	signal.Notify(s.termSig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
}

func (s *Service) setupInfo() {
	for _, c := range s.cps {
		info.AppendComponent(c.Info())
	}
}

// Run starts up all service components and monitors for errors.
// If a component returns a error the service is responsible for shutting down
// all components and terminate itself.
func (s *Service) Run() error {
	defer func() {
		err := trace.Close()
		if err != nil {
			log.Errorf("failed to close trace %v", err)
		}
	}()
	ctx, cnl := context.WithCancel(context.Background())
	chErr := make(chan error, len(s.cps))
	wg := sync.WaitGroup{}
	wg.Add(len(s.cps))
	for _, cp := range s.cps {
		go func(c Component) {
			defer wg.Done()
			chErr <- c.Run(ctx)
		}(cp)
	}

	var ee []error
	ee = append(ee, s.waitTermination(chErr))
	cnl()

	wg.Wait()
	close(chErr)

	for err := range chErr {
		ee = append(ee, err)
	}
	return errors.Aggregate(ee...)
}

// Setup set's up metrics and default logging.
func Setup(name, version string) error {

	lvl, ok := os.LookupEnv("PATRON_LOG_LEVEL")
	if !ok {
		lvl = string(log.InfoLevel)
	}

	info.UpsertConfig("log_level", lvl)
	hostname, err := os.Hostname()
	if err != nil {
		return errors.Wrap(err, "failed to get hostname")
	}
	info.UpdateHost(hostname)

	f := map[string]interface{}{
		"srv":  name,
		"ver":  version,
		"host": hostname,
	}
	logSetupOnce.Do(func() {
		err = log.Setup(zerolog.Create(log.Level(lvl)), f)
	})

	return err
}

func (s *Service) setupDefaultTracing(name, version string) error {
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
	info.UpsertConfig("jaeger-agent", agent)
	tp, ok := os.LookupEnv("PATRON_JAEGER_SAMPLER_TYPE")
	if !ok {
		tp = jaeger.SamplerTypeProbabilistic
	}
	info.UpsertConfig("jaeger-agent-sampler-type", tp)
	var prmVal = 0.0
	var prm = "0.0"

	if prm, ok := os.LookupEnv("PATRON_JAEGER_SAMPLER_PARAM"); ok {
		prmVal, err = strconv.ParseFloat(prm, 64)
		if err != nil {
			return errors.Wrap(err, "env var for jaeger sampler param is not valid")
		}
	}

	info.UpsertConfig("jaeger-agent-sampler-param", prm)
	log.Infof("setting up default tracing %s, %s with param %s", agent, tp, prm)
	return trace.Setup(name, version, agent, tp, prmVal)
}

func (s *Service) createHTTPComponent() (Component, error) {
	var err error
	var portVal = int64(50000)
	port, ok := os.LookupEnv("PATRON_HTTP_DEFAULT_PORT")
	if ok {
		portVal, err = strconv.ParseInt(port, 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "env var for HTTP default port is not valid")
		}
	}
	port = strconv.FormatInt(portVal, 10)
	log.Infof("creating default HTTP component at port %s", port)

	options := []http.OptionFunc{
		http.Port(int(portVal)),
	}

	if s.hcf != nil {
		options = append(options, http.HealthCheck(s.hcf))
	}

	if s.routes != nil {
		options = append(options, http.Routes(s.routes))
	}

	if s.middlewares != nil && len(s.middlewares) > 0 {
		options = append(options, http.Middlewares(s.middlewares...))
	}

	cp, err := http.New(options...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create default HTTP component")
	}

	return cp, nil
}

func (s *Service) waitTermination(chErr <-chan error) error {
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
