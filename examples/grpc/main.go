package main

import (
	"context"
	"fmt"
	"os"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/grpc"
	"github.com/beatlabs/patron/component/grpc/greeter"
	"github.com/beatlabs/patron/log"
)

func init() {
	err := os.Setenv("PATRON_LOG_LEVEL", "debug")
	if err != nil {
		fmt.Printf("failed to set log level env var: %v", err)
		os.Exit(1)
	}
	err = os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "1.0")
	if err != nil {
		fmt.Printf("failed to set sampler env vars: %v", err)
		os.Exit(1)
	}
	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", "50005")
	if err != nil {
		fmt.Printf("failed to set default patron port env vars: %v", err)
		os.Exit(1)
	}
}

type greeterServer struct {
	greeter.UnimplementedGreeterServer
}

func (gs *greeterServer) SayHello(ctx context.Context, req *greeter.HelloRequest) (*greeter.HelloReply, error) {
	log.FromContext(ctx).Infof("request received: %v", req.String())

	return &greeter.HelloReply{Message: fmt.Sprintf("Hello, %s %s!", req.GetFirstname(), req.GetLastname())}, nil
}

func main() {
	name := "grpc"
	version := "1.0.0"

	service, err := patron.New(name, version, patron.TextLogger())
	if err != nil {
		fmt.Printf("failed to set up service: %v", err)
		os.Exit(1)
	}

	cmp, err := grpc.New(50006).Create()
	if err != nil {
		log.Fatalf("failed to create gRPC component: %v", err)
	}

	greeter.RegisterGreeterServer(cmp.Server(), &greeterServer{})

	ctx := context.Background()
	err = service.WithComponents(cmp).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service: %v", err)
	}
}
