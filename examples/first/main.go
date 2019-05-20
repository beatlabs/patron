package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/thebeatapp/patron"
	"github.com/thebeatapp/patron/encoding/protobuf"
	"github.com/thebeatapp/patron/examples"
	"github.com/thebeatapp/patron/log"
	"github.com/thebeatapp/patron/sync"
	patronhttp "github.com/thebeatapp/patron/sync/http"
	tracehttp "github.com/thebeatapp/patron/trace/http"
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

	err := patron.Setup(name, version)
	if err != nil {
		fmt.Printf("failed to set up logging: %v", err)
		os.Exit(1)
	}

	// Set up routes
	routes := []patronhttp.Route{
		patronhttp.NewPostRoute("/", first, true),
	}

	// Setup a simple CORS middleware
	middlewareCors := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Add("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type")
			w.Header().Add("Access-Control-Allow-Credentials", "Allow")
			h(w, r)
		}
	}
	sig := patron.SIGHUP(func() {
		fmt.Println("exit gracefully...")
		os.Exit(0)
	})

	srv, err := patron.New(
		name,
		version,
		patron.Routes(routes),
		patron.Middlewares(middlewareCors),
		sig,
	)
	if err != nil {
		log.Fatalf("failed to create service %v", err)
	}

	err = srv.Run()
	if err != nil {
		log.Fatalf("failed to run service %v", err)
	}
}

func first(ctx context.Context, req *sync.Request) (*sync.Response, error) {

	var u examples.User

	err := req.Decode(&u)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}

	b, err := protobuf.Encode(&u)
	if err != nil {
		return nil, errors.Wrap(err, "failed create request")
	}

	secondRouteReq, err := http.NewRequest("GET", "http://localhost:50001", bytes.NewReader(b))
	if err != nil {
		return nil, errors.Wrap(err, "failed create request")
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
		return nil, errors.Wrap(err, "failed to post to second service")
	}

	log.FromContext(ctx).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return sync.NewResponse(fmt.Sprintf("got %s from second HTTP route", rsp.Status)), nil
}
