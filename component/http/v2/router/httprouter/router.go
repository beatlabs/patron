package httprouter

import (
	"errors"

	"github.com/julienschmidt/httprouter"
)

// OptionFunc definition to allow functional configuration of the router.
type OptionFunc func(*Config) error

// Config definition.
type Config struct {
	aliveCheckFunc AliveCheckFunc
	readyCheckFunc ReadyCheckFunc
	routes         []*Route
}

func New(oo ...OptionFunc) (*httprouter.Router, error) {
	cfg := &Config{
		aliveCheckFunc: func() AliveStatus { return Alive },
		readyCheckFunc: func() ReadyStatus { return Ready },
	}

	for _, option := range oo {
		err := option(cfg)
		if err != nil {
			return nil, err
		}
	}

	var stdRoutes []*Route

	mux := httprouter.New()
	stdRoutes = append(stdRoutes, metricRoute())
	stdRoutes = append(stdRoutes, profilingRoutes()...)
	stdRoutes = append(stdRoutes, aliveCheckRoute(cfg.aliveCheckFunc))
	stdRoutes = append(stdRoutes, readyCheckRoute(cfg.readyCheckFunc))

	for _, route := range stdRoutes {
		// TODO: handle middlewares.
		mux.HandlerFunc(route.method, route.path, route.handler)
	}

	for _, route := range cfg.routes {
		// TODO: handle middlewares.
		mux.HandlerFunc(route.method, route.path, route.handler)
	}

	return mux, nil
}

// Routes option for providing routes to the router.
func Routes(routes ...*Route) OptionFunc {
	return func(cfg *Config) error {
		if len(routes) == 0 {
			return errors.New("routes are empty")
		}
		cfg.routes = routes
		return nil
	}
}

// AliveCheck option for the router.
func AliveCheck(acf AliveCheckFunc) OptionFunc {
	return func(cfg *Config) error {
		if acf == nil {
			return errors.New("alive check function is nil")
		}
		cfg.aliveCheckFunc = acf
		return nil
	}
}

// ReadyCheck option for the router.
func ReadyCheck(rcf ReadyCheckFunc) OptionFunc {
	return func(cfg *Config) error {
		if rcf == nil {
			return errors.New("ready check function is nil")
		}
		cfg.readyCheckFunc = rcf
		return nil
	}
}
