package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/grpc"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
)

type greeterServer struct {
	examples.UnimplementedGreeterServer
}

func (gs *greeterServer) SayHello(ctx context.Context, req *examples.HelloRequest) (*examples.HelloReply, error) {
	log.FromContext(ctx).Infof("gRPC request received: %v", req.String())

	return &examples.HelloReply{Message: fmt.Sprintf("Hello, %s %s!", req.GetFirstname(), req.GetLastname())}, nil
}

func createGrpcServer() (patron.Component, error) {
	port, err := strconv.Atoi(examples.GRPCPort)
	if err != nil {
		return nil, errors.New("failed to convert grpc port")
	}

	cmp, err := grpc.New(port)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC component: %v", err)
	}

	examples.RegisterGreeterServer(cmp.Server(), &greeterServer{})

	return cmp, nil
}
