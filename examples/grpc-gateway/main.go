package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/grpc"
	grpcgateway "github.com/beatlabs/patron/component/grpc-gateway"
	"github.com/beatlabs/patron/examples/grpc-gateway/greeter"
	"github.com/beatlabs/patron/log"
)

func main() {
	name := "grpc-gateway"
	version := "1.0.0"

	err := patron.SetupLogging(name, version)
	if err != nil {
		fmt.Printf("failed to set up logging: %v", err)
		os.Exit(1)
	}

	srv := &greeter.UnimplementedGreeterServer{}

	grpcComponent, err := grpc.New(50006).Create()
	if err != nil {
		log.Fatalf("failed to create gRPC component: %v", err)
	}

	greeter.RegisterGreeterServer(grpcComponent.Server(), srv)

	cors := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Add("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type")
			w.Header().Add("Access-Control-Allow-Credentials", "Allow")
			h.ServeHTTP(w, r)
		})
	}

	grpcGwComponent, err := grpcgateway.New(50007).WithMiddleware(cors).Create()
	if err != nil {
		log.Fatalf("failed to create gRPC gateway component: %v", err)
	}

	ctx := context.Background()

	greeter.RegisterGreeterHandlerServer(ctx, grpcGwComponent.ServeMux(), srv)

	err = patron.New(name, version).WithComponents(grpcComponent, grpcGwComponent).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service: %v", err)
	}
}
