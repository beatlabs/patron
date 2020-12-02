package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/beatlabs/patron"
	patronhttp "github.com/beatlabs/patron/component/http"
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
}

// In the following example, we define a route that serves some random data.
// We call this route with and without Accept-Encoding headers so we that we test the compression methods
// $ curl -s localhost:50000/foo | wc -c
// 1398106
// $ curl -s localhost:50000/foo -H "Accept-Encoding: nonexisting" | wc -c
// 1398106
// $ curl -s localhost:50000/foo -H "Accept-Encoding: gzip" | wc -c
// 1053068
// $ curl -s localhost:50000/foo -H "Accept-Encoding: deflate" | wc -c
// 1053045
//
// The middleware is skipped for 'uncompressed' routes, as well as /metrics, /alive and /ready
// even if we define the relevant headers
// $ curl -s localhost:50000/bar -H "Accept-Encoding: gzip" | wc -c
// 1398106
// $ curl -s localhost:50000/metrics -H "Accept-Encoding: deflate"
//
func main() {
	name := "compression-middleware"
	version := "1.0.0"

	service, err := patron.New(name, version)
	handle(err)

	routesBuilder := patronhttp.NewRoutesBuilder().
		Append(patronhttp.NewRouteBuilder("/foo", rnd).MethodGet()).
		Append(patronhttp.NewRouteBuilder("/bar", rnd).MethodGet()).
		Append(patronhttp.NewRouteBuilder("/hello", hello).MethodGet())

	ctx := context.Background()
	err = service.
		WithRoutesBuilder(routesBuilder).
		WithUncompressedPaths("/bar").
		Run(ctx)
	handle(err)
}

// creates some random data to send back
func rnd(_ context.Context, _ *patronhttp.Request) (*patronhttp.Response, error) {
	rand.Seed(time.Now().UnixNano())
	data := make([]byte, 1<<20)
	_, err := rand.Read(data)
	if err != nil {
		return nil, err
	}

	return patronhttp.NewResponse(data), nil
}

func hello(_ context.Context, _ *patronhttp.Request) (*patronhttp.Response, error) {
	return patronhttp.NewResponse("hello!"), nil
}

func handle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
