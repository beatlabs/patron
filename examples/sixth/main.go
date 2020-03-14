package main

import (
	"context"
	"fmt"
	"os"

	"github.com/beatlabs/patron/cache/lru"

	"github.com/beatlabs/patron/encoding/json"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/sync"
	patronhttp "github.com/beatlabs/patron/sync/http"
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
	name := "sixth"
	version := "1.0.0"

	// Set up routes
	cache, err := lru.New(100)
	if err != nil {
		log.Fatalf("failed to init the cache %v", err)
	}
	cachedRoute := patronhttp.NewCachedRouteBuilder("/", sixth).WithCache(cache).MethodGet()

	ctx := context.Background()
	err = patron.New(name, version).WithRoutesBuilder(patronhttp.NewRoutesBuilder().Append(cachedRoute)).Run(ctx)
	if err != nil {
		log.Fatalf("failed to run patron service %v", err)
	}

}

func sixth(ctx context.Context, req *sync.Request) (*sync.Response, error) {

	var u examples.User
	println(fmt.Sprintf("u = %v", u))
	err := req.Decode(&u)
	println(fmt.Sprintf("err = %v", err))
	if err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)
	}

	b, err := json.Encode(&u)
	if err != nil {
		return nil, fmt.Errorf("failed create request: %w", err)
	}

	log.FromContext(ctx).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return sync.NewResponse(string(b)), nil
}
