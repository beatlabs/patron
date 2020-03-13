package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/sync"
	patronhttp "github.com/beatlabs/patron/sync/http"
	tracehttp "github.com/beatlabs/patron/trace/http"
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
}

func main() {
	name := "first"
	version := "1.0.0"

	err := patron.SetupLogging(name, version)
	if err != nil {
		fmt.Printf("failed to set up logging: %v", err)
		os.Exit(1)
	}

	routesBuilder := patronhttp.NewRoutesBuilder().Append(patronhttp.NewRouteBuilder("/", first).MethodPost())

	// Setup a simple CORS middleware
	middlewareCors := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Add("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type")
			w.Header().Add("Access-Control-Allow-Credentials", "Allow")
			h.ServeHTTP(w, r)
		})
	}
	sig := func() {
		fmt.Println("exit gracefully...")
		os.Exit(0)
	}

	ctx := context.Background()
	err = patron.New(name, version).
		WithRoutesBuilder(routesBuilder).
		WithMiddlewares(middlewareCors).
		WithSIGHUP(sig).
		Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}

func first(ctx context.Context, req *sync.Request) (*sync.Response, error) {

	var u examples.User

	err := req.Decode(&u)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)
	}

	b, err := protobuf.Encode(&u)
	if err != nil {
		return nil, fmt.Errorf("failed create request: %w", err)
	}

	secondRouteReq, err := http.NewRequest("GET", "http://localhost:50001", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed create request: %w", err)
	}
	secondRouteReq.Header.Add("Content-Type", protobuf.Type)
	secondRouteReq.Header.Add("Accept", protobuf.Type)
	secondRouteReq.Header.Add("Authorization", "Apikey 123456")
	cl, err := tracehttp.New(tracehttp.Timeout(5 * time.Second))
	if err != nil {
		return nil, err
	}
	rsp, err := cl.Do(ctx, secondRouteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to post to second service: %w", err)
	}

	log.FromContext(ctx).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return sync.NewResponse(fmt.Sprintf("got %s from second HTTP route", rsp.Status)), nil
}
