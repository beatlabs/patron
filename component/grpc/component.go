// Package grpc provides a gRPC component with included observability.
package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Component of a gRPC service.
type Component struct {
	port             int
	serverOptions    []grpc.ServerOption
	enableReflection bool
	srv              *grpc.Server
}

func New(port int, options ...OptionFunction) (*Component, error) {
	c := new(Component)
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("port is invalid: %d", port)
	}
	c.port = port

	var err error

	for _, optionFunc := range options {
		err = optionFunc(c)
		if err != nil {
			return nil, err
		}
	}

	c.serverOptions = append(c.serverOptions, grpc.UnaryInterceptor(observableUnaryInterceptor),
		grpc.StreamInterceptor(observableStreamInterceptor))
	srv := grpc.NewServer(c.serverOptions...)

	hs := health.NewServer()
	grpc_health_v1.RegisterHealthServer(srv, hs)

	if c.enableReflection {
		reflection.Register(srv)
	}

	c.srv = srv

	return c, nil
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

	slog.Debug("gRPC component listening", slog.Int("port", c.port))
	return c.srv.Serve(lis)
}
