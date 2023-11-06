package patron

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	patronErrors "github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/trace"
	"github.com/uber/jaeger-client-go"
)

const (
	srv  = "srv"
	ver  = "ver"
	host = "host"
)

// Component interface for implementing Service components.
type Component interface {
	Run(ctx context.Context) error
}

// Service is responsible for managing and setting up everything.
// The Service will start by default an HTTP component in order to host management endpoint.
type Service struct {
	name          string
	version       string
	termSig       chan os.Signal
	sighupHandler func()
	logConfig     logConfig
}

func New(name, version string, options ...OptionFunc) (*Service, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}
	if version == "" {
		version = "dev"
	}

	s := &Service{
		name:    name,
		version: version,
		termSig: make(chan os.Signal, 1),
		sighupHandler: func() {
			slog.Debug("sighup received: nothing setup")
		},
		logConfig: logConfig{
			attrs: defaultLogAttrs(name, version),
			json:  false,
		},
	}

	var err error
	err = setupJaegerTracing(name, version)
	if err != nil {
		return nil, err
	}

	optionErrors := make([]error, 0)
	for _, option := range options {
		err = option(s)
		if err != nil {
			optionErrors = append(optionErrors, err)
		}
	}

	if len(optionErrors) > 0 {
		return nil, patronErrors.Aggregate(optionErrors...)
	}

	setupLogging(s.logConfig)
	s.setupOSSignal()

	return s, nil
}

func (s *Service) Run(ctx context.Context, components ...Component) error {
	if len(components) == 0 || components[0] == nil {
		return errors.New("components are empty or nil")
	}

	defer func() {
		err := trace.Close()
		if err != nil {
			slog.Error("failed to close trace", slog.Any("error", err))
		}
	}()
	ctx, cnl := context.WithCancel(ctx)
	chErr := make(chan error, len(components))
	wg := sync.WaitGroup{}
	wg.Add(len(components))
	for _, cp := range components {
		go func(c Component) {
			defer wg.Done()
			chErr <- c.Run(ctx)
		}(cp)
	}

	log.FromContext(ctx).Info("service started", slog.String("name", s.name))
	ee := make([]error, 0, len(components))
	ee = append(ee, s.waitTermination(chErr))
	cnl()

	wg.Wait()
	close(chErr)

	for err := range chErr {
		ee = append(ee, err)
	}
	return patronErrors.Aggregate(ee...)
}

func (s *Service) setupOSSignal() {
	signal.Notify(s.termSig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
}

func (s *Service) waitTermination(chErr <-chan error) error {
	for {
		select {
		case sig := <-s.termSig:
			slog.Info("signal received", slog.Any("type", sig))

			switch sig {
			case syscall.SIGHUP:
				s.sighupHandler()
				return nil
			default:
				return nil
			}
		case err := <-chErr:
			if err != nil {
				slog.Info("component error received")
			}
			return err
		}
	}
}

type logConfig struct {
	attrs []slog.Attr
	json  bool
}

func getLogLevel() slog.Level {
	lvl, ok := os.LookupEnv("PATRON_LOG_LEVEL")
	if !ok {
		return slog.LevelInfo
	}

	lv := slog.LevelVar{}
	if err := lv.UnmarshalText([]byte(lvl)); err != nil {
		return slog.LevelInfo
	}

	return lv.Level()
}

func defaultLogAttrs(name, version string) []slog.Attr {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = host
	}

	return []slog.Attr{
		slog.String(srv, name),
		slog.String(ver, version),
		slog.String(host, hostname),
	}
}

func setupLogging(lc logConfig) {
	ho := &slog.HandlerOptions{
		AddSource: true,
		Level:     getLogLevel(),
	}

	var hnd slog.Handler

	if lc.json {
		hnd = slog.NewJSONHandler(os.Stderr, ho)
	} else {
		hnd = slog.NewTextHandler(os.Stderr, ho)
	}

	slog.New(hnd.WithAttrs(lc.attrs))
}

func setupJaegerTracing(name, version string) error {
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

	if prm, ok := os.LookupEnv("PATRON_JAEGER_SAMPLER_PARAM"); ok {
		tmpVal, err := strconv.ParseFloat(prm, 64)
		if err != nil {
			return fmt.Errorf("env var for jaeger sampler param is not valid: %w", err)
		}
		prmVal = tmpVal
	}

	var buckets []float64
	if b, ok := os.LookupEnv("PATRON_JAEGER_DEFAULT_BUCKETS"); ok {
		for _, bs := range strings.Split(b, ",") {
			val, err := strconv.ParseFloat(strings.TrimSpace(bs), 64)
			if err != nil {
				return fmt.Errorf("env var for jaeger default buckets contains invalid value: %w", err)
			}
			buckets = append(buckets, val)
		}
	}

	slog.Debug("setting up default tracing", slog.String("agent", agent), slog.String("param", tp), slog.Float64("val", prmVal))
	return trace.Setup(name, version, agent, tp, prmVal, buckets)
}
