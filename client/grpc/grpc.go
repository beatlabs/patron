// Package grpc provides a client implementation for gRPC with tracing and metrics included.
package grpc

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// DialContext creates a client connection to the given target with a context and
// a tracing and metrics unary interceptor.
func NewClient(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	if len(opts) == 0 {
		opts = make([]grpc.DialOption, 0)
	}

	opts = append(opts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))

	return grpc.NewClient(target, opts...)
}
