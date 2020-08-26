package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/cache/redis"
	patronhttp "github.com/beatlabs/patron/component/http"
	httpcache "github.com/beatlabs/patron/component/http/cache"
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
	err = os.Setenv("PATRON_HTTP_DEFAULT_PORT", "50006")
	if err != nil {
		fmt.Printf("failed to set default patron port env vars: %v", err)
		os.Exit(1)
	}
}

func main() {
	name := "seventh"
	version := "1.0.0"

	err := patron.SetupLogging(name, version)
	if err != nil {
		fmt.Printf("failed to set up logging: %v", err)
		os.Exit(1)
	}

	ctx := context.Background()

	cache, err := redis.New(ctx, redis.Options{})
	if err != nil {
		fmt.Printf("failed to set up redis cache: %v", err)
		os.Exit(1)
	}

	routesBuilder := patronhttp.NewRoutesBuilder().
		Append(patronhttp.NewRouteBuilder("/", seventh).
			WithRouteCache(cache, httpcache.Age{
				// we wont allow to override the cache more than once per 15 seconds
				Min: 15 * time.Second,
				// by default we might send stale response for up to 1 minute
				Max: 60 * time.Second,
			}).
			MethodGet())

	sig := func() {
		fmt.Println("exit gracefully...")
		os.Exit(0)
	}

	err = patron.New(name, version).
		WithRoutesBuilder(routesBuilder).
		WithSIGHUP(sig).
		Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}

// seventh gives the 7 minute interval of the current unix timestamp
// since the response will be the same for the next 7 minutes, it s a good use-case to apply caching
func seventh(ctx context.Context, req *patronhttp.Request) (*patronhttp.Response, error) {
	now := time.Now()
	minutes := now.Unix() / 60
	minuteInterval := minutes / 7
	return patronhttp.NewResponse(fmt.Sprintf("current unix 7-minute interval is (%d) called at %v", minuteInterval, now.Unix())), nil
}
