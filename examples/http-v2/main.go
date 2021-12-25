package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/component/http/v2/router/httprouter"
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

	route, err := httprouter.NewRoute(http.MethodGet, "/api/search", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(writer, "articles")
	})

	router, err := httprouter.New(httprouter.Routes(route))
	if err != nil {
		log.Fatalf("failed to router %v", err)
	}

	ctx := context.Background()
	err = service.WithRouter(router).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}
