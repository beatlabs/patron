package patron

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/beatlabs/patron/observability"
	"github.com/beatlabs/patron/observability/log"
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
	name                  string
	version               string
	termSig               chan os.Signal
	sighupHandler         func()
	observabilityCfg      observability.Config
	observabilityProvider *observability.Provider
	shutdownTimeout       time.Duration
}

// New creates a new Service instance.
func New(ctx context.Context, name, version string, options ...OptionFunc) (*Service, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}
	if version == "" {
		version = "dev"
	}

	if ctx == nil {
		slog.Warn("nil context provided, using background context")
		ctx = context.Background()
	}

	cfg := observabilityConfig(name, version)

	observabilityProvider, err := observability.Setup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	s := &Service{
		name:    name,
		version: version,
		termSig: make(chan os.Signal, 1),
		sighupHandler: func() {
			slog.Debug("sighup received: nothing setup")
		},
		observabilityCfg:      cfg,
		observabilityProvider: observabilityProvider,
		shutdownTimeout:       5 * time.Second, // default timeout
	}

	optionErrors := make([]error, 0)

	for _, option := range options {
		err = option(s)
		if err != nil {
			optionErrors = append(optionErrors, err)
		}
	}

	if len(optionErrors) > 0 {
		return nil, errors.Join(optionErrors...)
	}

	s.setupOSSignal()

	return s, nil
}

// Run starts the service with the provided components.
func (s *Service) Run(ctx context.Context, components ...Component) error {
	if len(components) == 0 {
		return errors.New("no components provided")
	}

	for i, comp := range components {
		if comp == nil {
			return errors.New("component at index " + string(rune('0'+i)) + " is nil")
		}
	}

	defer func() {
		ctx, cnl := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cnl()

		err := s.observabilityProvider.Shutdown(ctx)
		if err != nil {
			slog.Error("failed to close observability provider", log.ErrorAttr(err))
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
	return errors.Join(ee...)
}

// setupOSSignal configures the service to handle OS signals:
// - SIGTERM/Interrupt: Graceful shutdown of all components
// - SIGHUP: Configuration reload (if handler configured)
// The service will attempt to gracefully shutdown all components when a termination
// signal is received, using the configured shutdown timeout.
func (s *Service) setupOSSignal() {
	signal.Notify(s.termSig, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
}

// waitTermination waits for either a termination signal or a component error.
// Returns nil on graceful shutdown (SIGTERM/Interrupt/SIGHUP) or the first error
// encountered from any component.
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

func observabilityConfig(name, version string) observability.Config {
	var lvl string
	lvl, ok := os.LookupEnv("PATRON_LOG_LEVEL")
	if !ok {
		lvl = "info"
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = host
	}

	attrs := []slog.Attr{
		slog.String(srv, name),
		slog.String(ver, version),
		slog.String(host, hostname),
	}

	return observability.Config{
		Name:    name,
		Version: version,
		LogConfig: log.Config{
			Attributes: attrs,
			IsJSON:     false,
			Level:      lvl,
		},
	}
}
