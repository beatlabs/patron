package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/beatlabs/patron"
	clienthttp "github.com/beatlabs/patron/client/http"
	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/log/std"
)

func init() {
	err := os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "1.0")
	if err != nil {
		fmt.Printf("failed to set sampler env vars: %v", err)
		os.Exit(1)
	}
}

func main() {
	name := "httpHandler"
	version := "1.0.0"

	logger := std.New(os.Stderr, log.DebugLevel, map[string]interface{}{"env": "staging"})

	service, err := patron.New(name, version, patron.Logger(logger))
	if err != nil {
		fmt.Printf("failed to set up service: %v", err)
		os.Exit(1)
	}

	routesBuilder := patronhttp.NewRoutesBuilder().Append(patronhttp.NewPostRouteBuilder("/", httpHandler)).
		Append(patronhttp.NewGetRouteBuilder("/", getHandler).WithRateLimiting(50, 50))

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
		log.Info("exit gracefully...")
		os.Exit(0)
	}

	ctx := context.Background()
	err = service.
		WithRoutesBuilder(routesBuilder).
		WithMiddlewares(middlewareCors).
		WithSIGHUP(sig).
		Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}
func getHandler(ctx context.Context, req *patronhttp.Request) (*patronhttp.Response, error) {
	return patronhttp.NewResponse(fmt.Sprint("Testing Middleware", http.StatusOK)), nil
}

func httpHandler(ctx context.Context, req *patronhttp.Request) (*patronhttp.Response, error) {
	interval, err := DoIntervalRequest(ctx)
	if err != nil {
		log.FromContext(ctx).Infof("httpHandler: failed to get interval information %v: could it be that the http-cache service is not running ?", err)
	} else {
		log.FromContext(ctx).Infof("httpHandler: pipeline initiated at: %s", interval)
	}

	var u examples.User

	err = req.Decode(&u)
	if err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)
	}

	b, err := protobuf.Encode(&u)
	if err != nil {
		return nil, fmt.Errorf("failed create request: %w", err)
	}

	kafkaRouteReq, err := http.NewRequest("GET", "http://localhost:50001", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed create request: %w", err)
	}
	kafkaRouteReq.Header.Add("Content-Type", protobuf.Type)
	kafkaRouteReq.Header.Add("Accept", protobuf.Type)
	kafkaRouteReq.Header.Add("Authorization", "Apikey 123456")
	cl, err := clienthttp.New(clienthttp.Timeout(5 * time.Second))
	if err != nil {
		return nil, err
	}
	rsp, err := cl.Do(ctx, kafkaRouteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to post to kafka service: %w", err)
	}
	log.FromContext(ctx).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return patronhttp.NewResponse(fmt.Sprintf("got %s from kafka HTTP route", rsp.Status)), nil
}

// DoIntervalRequest is a helper method to make a request to the http-cache example service from other examples
func DoIntervalRequest(ctx context.Context) (string, error) {
	request, err := http.NewRequest("GET", "http://localhost:50007/", nil)
	if err != nil {
		return "", fmt.Errorf("failed create route request: %w", err)
	}
	cl, err := clienthttp.New(clienthttp.Timeout(5 * time.Second))
	if err != nil {
		return "", fmt.Errorf("could not create http client: %w", err)
	}

	response, err := cl.Do(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed create get to http-cache service: %w", err)
	}

	tb, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decode http-cache response body: %w", err)
	}

	rgx := regexp.MustCompile(`\((.*?)\)`)
	timeInstance := rgx.FindStringSubmatch(string(tb))
	if len(timeInstance) == 1 {
		return "", fmt.Errorf("could not match timeinstance from response %s", string(tb))
	}
	return timeInstance[1], nil
}
