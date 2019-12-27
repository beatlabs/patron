package grpc

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
)

// Component of a gRPC service.
type Component struct {
	port string
}

// Run the gRPC service.
func (c *Component) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", c.port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	//s.RegisterService(sd *grpc.ServiceDesc, ss interface{})

	go func() {
		for {
			select {
			case <-ctx.Done():
				s.GracefulStop()
			}
		}
	}()

	return s.Serve(lis)
}

// Builder pattern for our gRPC service.
type Builder struct {
}
