package grpc

import (
	"errors"

	"google.golang.org/grpc"
)

type OptionFunction func(*Component) error

// WithServerOptions allows gRPC server options to be set.
func WithServerOptions(options ...grpc.ServerOption) OptionFunction {
	return func(component *Component) error {
		if len(options) == 0 {
			return errors.New("no grpc options provided")
		}

		component.serverOptions = options
		return nil
	}
}

// WithReflection opt-in for gRPC reflection.
// WithReflection could be considered a security risk if services are exposed to public internet.
func WithReflection() OptionFunction {
	return func(component *Component) error {
		component.enableReflection = true
		return nil
	}
}
