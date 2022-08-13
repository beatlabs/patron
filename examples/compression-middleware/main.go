package main

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/beatlabs/patron"
	v2 "github.com/beatlabs/patron/component/http/v2"
	"github.com/beatlabs/patron/component/http/v2/router/httprouter"
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
func main() {
	name := "compression-middleware"
	version := "1.0.0"

	service, err := patron.New(name, version)
	handle(err)

	var routes v2.Routes
	routes.Append(v2.NewGetRoute("/foo", rnd))
	routes.Append(v2.NewGetRoute("/bar", rnd))
	routes.Append(v2.NewGetRoute("/hello", hello))
	rr, err := routes.Result()
	if err != nil {
		log.Fatalf("failed to create routes: %v", err)
	}

	router, err := httprouter.New(httprouter.Routes(rr...))
	if err != nil {
		log.Fatalf("failed to create http router: %v", err)
	}

	ctx := context.Background()
	err = service.WithRouter(router).Run(ctx)
	handle(err)
}

func rnd(rw http.ResponseWriter, _ *http.Request) {
	rand.Seed(time.Now().UnixNano())
	data := make([]byte, 1<<20)
	_, err := rand.Read(data)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusCreated)
	_, _ = rw.Write(data)
	return
}

func hello(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusCreated)
	_, _ = rw.Write([]byte("hello!"))
	return
}

func handle(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
