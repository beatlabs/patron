package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/log"
	"github.com/gorilla/mux"
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

	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", "50001")
	if err != nil {
		fmt.Printf("failed to set default patron port env vars: %v", err)
		os.Exit(1)
	}
}

var version = "1.0.0"

func main() {
	service, err := patron.New("http-v2", version, patron.TextLogger())
	if err != nil {
		fmt.Printf("failed to set up service: %v", err)
		os.Exit(1)
	}

	router := mux.NewRouter()
	router.NewRoute()
	router.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(rw, "Home")
	}) //.Subrouter().Use(mwf ...mux.MiddlewareFunc)

	ctx := context.Background()
	err = service.WithMuxRouter(router).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}
