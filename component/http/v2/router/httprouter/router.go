package httprouter

import (
	"errors"
	"fmt"
	"os"

	"github.com/beatlabs/patron/component/http/middleware"
	"github.com/beatlabs/patron/component/http/v2"
	"github.com/beatlabs/patron/log"
	"github.com/julienschmidt/httprouter"
)

const defaultDeflateLevel = 6

// OptionFunc definition to allow functional configuration of the router.
type OptionFunc func(*Config) error

// Config definition.
type Config struct {
	aliveCheckFunc        v2.LivenessCheckFunc
	readyCheckFunc        v2.ReadyCheckFunc
	deflateLevel          int
	middlewares           []middleware.Func
	routes                []*v2.Route
	enableProfilingExpVar bool
}

func New(oo ...OptionFunc) (*httprouter.Router, error) {
	cfg := &Config{
		aliveCheckFunc: func() v2.AliveStatus { return v2.Alive },
		readyCheckFunc: func() v2.ReadyStatus { return v2.Ready },
		deflateLevel:   defaultDeflateLevel,
	}

	for _, option := range oo {
		err := option(cfg)
		if err != nil {
			return nil, err
		}
	}

	var stdRoutes []*v2.Route

	mux := httprouter.New()
	stdRoutes = append(stdRoutes, v2.MetricRoute())
	stdRoutes = append(stdRoutes, v2.ProfilingRoutes(cfg.enableProfilingExpVar)...)

	route, err := v2.LivenessCheckRoute(cfg.aliveCheckFunc)
	if err != nil {
		return nil, err
	}
	stdRoutes = append(stdRoutes, route)

	route, err = v2.ReadyCheckRoute(cfg.readyCheckFunc)
	if err != nil {
		return nil, err
	}
	stdRoutes = append(stdRoutes, route)

	for _, route := range stdRoutes {
		handler := middleware.Chain(route.Handler(), middleware.NewRecovery())
		mux.Handler(route.Method(), route.Path(), handler)
		log.Debugf("added route %s", route)
	}

	// parse a list of HTTP numeric status codes that must be logged
	statusCodeLoggerCfg, _ := os.LookupEnv("PATRON_HTTP_STATUS_ERROR_LOGGING")
	statusCodeLogger, err := middleware.NewStatusCodeLoggerHandler(statusCodeLoggerCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse status codes %s: %w", statusCodeLoggerCfg, err)
	}

	for _, route := range cfg.routes {
		// add standard middlewares
		middlewares := []middleware.Func{
			middleware.NewRecovery(),
			middleware.NewInjectObservability(),
			middleware.NewLoggingTracing(route.Path(), statusCodeLogger),
			middleware.NewRequestObserver(route.Method(), route.Path()),
			middleware.NewCompression(cfg.deflateLevel),
		}
		// add router middlewares
		middlewares = append(middlewares, cfg.middlewares...)
		// add route middlewares
		middlewares = append(middlewares, route.Middlewares()...)
		// chain all middlewares to the handler
		handler := middleware.Chain(route.Handler(), middlewares...)
		mux.Handler(route.Method(), route.Path(), handler)
		log.Debugf("added route %s with %d middlewares", route, len(middlewares))
	}

	return mux, nil
}

// Routes option for providing routes to the router.
func Routes(routes ...*v2.Route) OptionFunc {
	return func(cfg *Config) error {
		if len(routes) == 0 {
			return errors.New("routes are empty")
		}
		cfg.routes = routes
		return nil
	}
}

// AliveCheck option for the router.
func AliveCheck(acf v2.LivenessCheckFunc) OptionFunc {
	return func(cfg *Config) error {
		if acf == nil {
			return errors.New("alive check function is nil")
		}
		cfg.aliveCheckFunc = acf
		return nil
	}
}

// ReadyCheck option for the router.
func ReadyCheck(rcf v2.ReadyCheckFunc) OptionFunc {
	return func(cfg *Config) error {
		if rcf == nil {
			return errors.New("ready check function is nil")
		}
		cfg.readyCheckFunc = rcf
		return nil
	}
}

// DeflateLevel option for the compression middleware.
func DeflateLevel(level int) OptionFunc {
	return func(cfg *Config) error {
		cfg.deflateLevel = level
		return nil
	}
}

// Middlewares option for middlewares.
func Middlewares(mm ...middleware.Func) OptionFunc {
	return func(cfg *Config) error {
		if len(mm) == 0 {
			return errors.New("middlewares are empty")
		}
		cfg.middlewares = mm
		return nil
	}
}

// EnableExpVarProfiling option for enabling expVar in profiling endpoints.
func EnableExpVarProfiling() OptionFunc {
	return func(cfg *Config) error {
		cfg.enableProfilingExpVar = true
		return nil
	}
}
