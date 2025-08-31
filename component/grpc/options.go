package grpc

import (
	"errors"

	"google.golang.org/grpc"
)

// OptionFunction configures the gRPC Component.
type OptionFunction func(*Component) error

// WithServerOptions applies grpc.ServerOption values to the server.
func WithServerOptions(options ...grpc.ServerOption) OptionFunction {
	return func(component *Component) error {
		if len(options) == 0 {
			return errors.New("no grpc options provided")
		}

		component.serverOptions = options
		return nil
	}
}

// WithReflection enables server reflection. Be cautious when exposing to the public internet.
func WithReflection() OptionFunction {
	return func(component *Component) error {
		component.enableReflection = true
		return nil
	}
}
