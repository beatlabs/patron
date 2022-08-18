package httprouter

import (
	"errors"

	"github.com/beatlabs/patron/component/http/middleware"
	v2 "github.com/beatlabs/patron/component/http/v2"
)

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

// EnableAppNameHeaders option for adding name and version header to the response.
func EnableAppNameHeaders(name, version string) OptionFunc {
	return func(cfg *Config) error {
		if name == "" {
			return errors.New("app name was not provided")
		}

		if version == "" {
			return errors.New("app version was not provided")
		}

		cfg.appNameVersionMiddleware = middleware.NewAppNameVersion(name, version)
		return nil
	}
}
