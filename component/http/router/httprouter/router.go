package httprouter

import (
	"fmt"
	"log/slog"
	"os"

	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/component/http/middleware"
	"github.com/julienschmidt/httprouter"
)

const defaultDeflateLevel = 6

// OptionFunc definition to allow functional configuration of the router.
type OptionFunc func(*Config) error

// Config definition.
type Config struct {
	aliveCheckFunc           patronhttp.LivenessCheckFunc
	readyCheckFunc           patronhttp.ReadyCheckFunc
	deflateLevel             int
	middlewares              []middleware.Func
	routes                   []*patronhttp.Route
	enableProfilingExpVar    bool
	appNameVersionMiddleware middleware.Func
}

// New creates an http router with functional options.
func New(oo ...OptionFunc) (*httprouter.Router, error) {
	cfg := &Config{
		aliveCheckFunc: func() patronhttp.AliveStatus { return patronhttp.Alive },
		readyCheckFunc: func() patronhttp.ReadyStatus { return patronhttp.Ready },
		deflateLevel:   defaultDeflateLevel,
	}

	for _, option := range oo {
		err := option(cfg)
		if err != nil {
			return nil, err
		}
	}

	var stdRoutes []*patronhttp.Route

	mux := httprouter.New()
	stdRoutes = append(stdRoutes, patronhttp.MetricRoute())
	stdRoutes = append(stdRoutes, patronhttp.ProfilingRoutes(cfg.enableProfilingExpVar)...)

	route, err := patronhttp.LivenessCheckRoute(cfg.aliveCheckFunc)
	if err != nil {
		return nil, err
	}
	stdRoutes = append(stdRoutes, route)

	route, err = patronhttp.ReadyCheckRoute(cfg.readyCheckFunc)
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
		slog.Debug("added route", slog.Any("route", route))
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
		loggingTracingMiddleware, err := middleware.NewLoggingTracing(route.Path(), statusCodeLogger)
		if err != nil {
			return nil, err
		}
		middlewares = append(middlewares, loggingTracingMiddleware)
		requestObserverMiddleware, err := middleware.NewRequestObserver(route.Method(), route.Path())
		if err != nil {
			return nil, err
		}
		middlewares = append(middlewares, requestObserverMiddleware)
		compressionMiddleware, err := middleware.NewCompression(cfg.deflateLevel)
		if err != nil {
			return nil, err
		}
		middlewares = append(middlewares, compressionMiddleware)

		// add router middlewares
		middlewares = append(middlewares, cfg.middlewares...)
		// add route middlewares
		middlewares = append(middlewares, route.Middlewares()...)
		// chain all middlewares to the handler
		handler := middleware.Chain(route.Handler(), middlewares...)
		mux.Handler(route.Method(), route.Path(), handler)
		slog.Debug("added route with middlewares", slog.Any("route", route), slog.Int("middlewares", len(middlewares)))
	}

	return mux, nil
}
