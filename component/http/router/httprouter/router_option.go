package httprouter

import (
	"errors"

	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/component/http/middleware"
)

// WithRoutes option for providing routes to the router.
func WithRoutes(routes ...*patronhttp.Route) OptionFunc {
	return func(cfg *Config) error {
		if len(routes) == 0 {
			return errors.New("routes are empty")
		}
		cfg.routes = routes
		return nil
	}
}

// WithAliveCheck option for the router.
func WithAliveCheck(acf patronhttp.LivenessCheckFunc) OptionFunc {
	return func(cfg *Config) error {
		if acf == nil {
			return errors.New("alive check function is nil")
		}
		cfg.aliveCheckFunc = acf
		return nil
	}
}

// WithReadyCheck option for the router.
func WithReadyCheck(rcf patronhttp.ReadyCheckFunc) OptionFunc {
	return func(cfg *Config) error {
		if rcf == nil {
			return errors.New("ready check function is nil")
		}
		cfg.readyCheckFunc = rcf
		return nil
	}
}

// WithDeflateLevel option for the compression middleware.
func WithDeflateLevel(level int) OptionFunc {
	return func(cfg *Config) error {
		if level < -2 || level > 9 {
			return errors.New("provided deflate level value not in the [-2, 9] range")
		}

		cfg.deflateLevel = level
		return nil
	}
}

// WithMiddlewares option for middlewares.
func WithMiddlewares(mm ...middleware.Func) OptionFunc {
	return func(cfg *Config) error {
		if len(mm) == 0 {
			return errors.New("middlewares are empty")
		}
		cfg.middlewares = mm
		return nil
	}
}

// WithExpVarProfiling option for enabling expVar in profiling endpoints.
func WithExpVarProfiling() OptionFunc {
	return func(cfg *Config) error {
		cfg.enableProfilingExpVar = true
		return nil
	}
}

// WithAppNameHeaders option for adding name and version header to the response.
func WithAppNameHeaders(name, version string) (OptionFunc, error) {
	appNameVersionMiddleware, err := middleware.NewAppNameVersion(name, version)
	if err != nil {
		return nil, err
	}
	return func(cfg *Config) error {
		cfg.appNameVersionMiddleware = appNameVersionMiddleware
		return nil
	}, nil
}
