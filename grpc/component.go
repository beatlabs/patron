package grpc

import (
	"context"
	"log"
	"net"

	"github.com/beatlabs/patron/errors"
	"google.golang.org/grpc"
)

// Component of a gRPC service.
type Component struct {
	port string
	srv  *grpc.Server
}

// Run the gRPC service.
func (c *Component) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", c.port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	//s.RegisterService(sd *grpc.ServiceDesc, ss interface{})

	go func() {
		for range ctx.Done() {
			c.srv.GracefulStop()
			break
		}
	}()

	return c.srv.Serve(lis)
}

// Builder pattern for our gRPC service.
type Builder struct {
	port   string
	errors []error
}

// New builder.
func New(port string) *Builder {
	b := &Builder{}
	if port == "" {
		b.errors = append(b.errors, errors.New("port is empty"))
	}
	b.port = port
	return b
}

// Create the gRPC component.
func (b *Builder) Create() (*Component, error) {
	if len(b.errors) != 0 {
		return nil, errors.Aggregate(b.errors...)
	}
	// TODO: create grpc server and options...
	srv := grpc.NewServer()
	return &Component{port: b.port, srv: srv}, nil
}
