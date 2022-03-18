package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/cache/redis"
	httpcache "github.com/beatlabs/patron/component/http/cache"
	v2 "github.com/beatlabs/patron/component/http/v2"
	"github.com/beatlabs/patron/component/http/v2/router/httprouter"
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
	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", "50007")
	if err != nil {
		fmt.Printf("failed to set default patron port env vars: %v", err)
		os.Exit(1)
	}
}

func main() {
	name := "http-cache"
	version := "1.0.0"

	service, err := patron.New(name, version, patron.TextLogger())
	if err != nil {
		fmt.Printf("failed to set up service: %v", err)
		os.Exit(1)
	}

	ctx := context.Background()

	cache, err := redis.New(ctx, redis.Options{})
	if err != nil {
		fmt.Printf("failed to set up redis cache: %v", err)
		os.Exit(1)
	}

	var routes v2.Routes
	routes.Append(v2.NewGetRoute("/", handler, v2.Cache(cache, httpcache.Age{
		// we won't allow to override the cache more than once per 15 seconds
		Min: 15 * time.Second,
		// by default, we might send stale response for up to 1 minute
		Max: 60 * time.Second,
	})))
	rr, err := routes.Result()
	if err != nil {
		log.Fatalf("failed to create routes: %v", err)
	}

	router, err := httprouter.New(httprouter.Routes(rr...))
	if err != nil {
		log.Fatalf("failed to create http router: %v", err)
	}

	sig := func() {
		fmt.Println("exit gracefully...")
		os.Exit(0)
	}

	err = service.WithRouter(router).WithSIGHUP(sig).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}

// handler gives the 7-minute interval of the current unix timestamp
// since the response will be the same for the next 7 minutes, it s a good use-case to apply caching
func handler(rw http.ResponseWriter, _ *http.Request) {
	now := time.Now()
	minutes := now.Unix() / 60
	minuteInterval := minutes / 7
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte(fmt.Sprintf("current unix 7-minute interval is (%d) called at %v", minuteInterval, now.Unix())))
}
