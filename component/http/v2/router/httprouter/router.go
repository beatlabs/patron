package httprouter

import (
	"fmt"
	"os"

	"github.com/beatlabs/patron/component/http/middleware"
	v2 "github.com/beatlabs/patron/component/http/v2"
	"github.com/beatlabs/patron/log"
	"github.com/julienschmidt/httprouter"
)

const defaultDeflateLevel = 6

// OptionFunc definition to allow functional configuration of the router.
type OptionFunc func(*Config) error

// Config definition.
type Config struct {
	aliveCheckFunc           v2.LivenessCheckFunc
	readyCheckFunc           v2.ReadyCheckFunc
	deflateLevel             int
	middlewares              []middleware.Func
	routes                   []*v2.Route
	enableProfilingExpVar    bool
	appNameVersionMiddleware middleware.Func
}

// New creates an http router with functional options.
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

	stdMiddlewares := []middleware.Func{middleware.NewRecovery()}
	if cfg.appNameVersionMiddleware != nil {
		stdMiddlewares = append(stdMiddlewares, cfg.appNameVersionMiddleware)
	}

	for _, route := range stdRoutes {
		handler := middleware.Chain(route.Handler(), stdMiddlewares...)
		mux.Handler(route.Method(), route.Path(), handler)
		log.Debugf("added route %s", route)
	}

	// parse a list of HTTP numeric status codes that must be logged
	statusCodeLoggerCfg, _ := os.LookupEnv("PATRON_HTTP_STATUS_ERROR_LOGGING")
	statusCodeLogger, err := middleware.NewStatusCodeLoggerHandler(statusCodeLoggerCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse status codes %s: %w", statusCodeLoggerCfg, err)
	}

	// add to the default middlewares the observability we need per route.
	stdMiddlewares = append(stdMiddlewares, middleware.NewInjectObservability())

	for _, route := range cfg.routes {
		var middlewares []middleware.Func
		middlewares = append(middlewares, stdMiddlewares...)
		middlewares = append(middlewares, middleware.NewLoggingTracing(route.Path(), statusCodeLogger))
		middlewares = append(middlewares, middleware.NewRequestObserver(route.Method(), route.Path()))
		middlewares = append(middlewares, middleware.NewCompression(cfg.deflateLevel))

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
