package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/grpc"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/observability/log"
)

type greeterServer struct {
	examples.UnimplementedGreeterServer
}

func (gs *greeterServer) SayHello(ctx context.Context, req *examples.HelloRequest) (*examples.HelloReply, error) {
	log.FromContext(ctx).Info("gRPC request received", slog.String("req", req.String()))

	return &examples.HelloReply{Message: fmt.Sprintf("Hello, %s %s!", req.GetFirstname(), req.GetLastname())}, nil
}

func createGrpcServer() (patron.Component, error) {
	port, err := strconv.Atoi(examples.GRPCPort)
	if err != nil {
		return nil, errors.New("failed to convert grpc port")
	}

	cmp, err := grpc.New(port)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC component: %w", err)
	}

	examples.RegisterGreeterServer(cmp.Server(), &greeterServer{})

	return cmp, nil
}
