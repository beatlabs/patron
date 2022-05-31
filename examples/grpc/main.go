package main

import (
	"context"
	"fmt"
	"os"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/grpc"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
)

func init() {
	err := os.Setenv("PATRON_LOG_LEVEL", "debug")
	if err != nil {
		log.Fatalf("failed to set log level env var: %v", err)
	}
	err = os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "1.0")
	if err != nil {
		log.Fatalf("failed to set sampler env vars: %v", err)
	}
	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", "50005")
	if err != nil {
		log.Fatalf("failed to set default patron port env vars: %v", err)
	}
}

type greeterServer struct {
	examples.UnimplementedGreeterServer
}

func (gs *greeterServer) SayHello(ctx context.Context, req *examples.HelloRequest) (*examples.HelloReply, error) {
	log.FromContext(ctx).Infof("request received: %v", req.String())

	return &examples.HelloReply{Message: fmt.Sprintf("Hello, %s %s!", req.GetFirstname(), req.GetLastname())}, nil
}

func main() {
	name := "grpc"
	version := "1.0.0"

	service, err := patron.New(name, version, patron.TextLogger())
	if err != nil {
		log.Fatalf("failed to set up service: %v", err)
	}

	cmp, err := grpc.New(50006).Create()
	if err != nil {
		log.Fatalf("failed to create gRPC component: %v", err)
	}

	examples.RegisterGreeterServer(cmp.Server(), &greeterServer{})

	ctx := context.Background()
	err = service.WithComponents(cmp).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service: %v", err)
	}
}
