package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/beatlabs/patron/errors"
	"github.com/beatlabs/patron/log"
	"google.golang.org/grpc"
)

// Component of a gRPC service.
type Component struct {
	port int
	srv  *grpc.Server
}

// Server returns the gRPC sever.
func (c *Component) Server() *grpc.Server {
	return c.srv
}

// Run the gRPC service.
func (c *Component) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", c.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		<-ctx.Done()
		c.srv.GracefulStop()
	}()

	log.Infof("gRPC component listening on port %d", c.port)
	return c.srv.Serve(lis)
}

// Builder pattern for our gRPC service.
type Builder struct {
	port          int
	serverOptions []grpc.ServerOption
	errors        []error
}

// New builder.
func New(port int) *Builder {
	b := &Builder{}
	if port <= 0 || port > 65535 {
		b.errors = append(b.errors, fmt.Errorf("port is invalid: %d", port))
		return b
	}
	b.port = port
	return b
}

// WithOptions allows gRPC server options to be set.
func (b *Builder) WithOptions(oo ...grpc.ServerOption) *Builder {
	if len(b.errors) != 0 {
		return b
	}
	b.serverOptions = append(b.serverOptions, oo...)
	return b
}

// Create the gRPC component.
func (b *Builder) Create() (*Component, error) {
	if len(b.errors) != 0 {
		return nil, errors.Aggregate(b.errors...)
	}

	b.serverOptions = append(b.serverOptions, grpc.UnaryInterceptor(observableUnaryInterceptor),
		grpc.StreamInterceptor(observableStreamInterceptor))

	srv := grpc.NewServer(b.serverOptions...)

	return &Component{
		port: b.port,
		srv:  srv,
	}, nil
}
