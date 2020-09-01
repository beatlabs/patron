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
		WithLogFields(map[string]interface{}{"env": "staging"}).
		WithSIGHUP(sig).
		Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}

func first(ctx context.Context, req *patronhttp.Request) (*patronhttp.Response, error) {

	timing, err := DoTimingRequest(ctx)
	if err != nil {
		log.FromContext(ctx).Infof("first: failed to get timing information %v: could it be that the seventh service is not running ?", err)
	} else {
		log.FromContext(ctx).Infof("first: pipeline initiated at: %s", timing)
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

	secondRouteReq, err := http.NewRequest("GET", "http://localhost:50001", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed create request: %w", err)
	}
	secondRouteReq.Header.Add("Content-Type", protobuf.Type)
	secondRouteReq.Header.Add("Accept", protobuf.Type)
	secondRouteReq.Header.Add("Authorization", "Apikey 123456")
	cl, err := clienthttp.New(clienthttp.Timeout(5 * time.Second))
	if err != nil {
		return nil, err
	}
	rsp, err := cl.Do(ctx, secondRouteReq)
	if err != nil {
		return nil, fmt.Errorf("failed to post to second service: %w", err)
	}
	log.FromContext(ctx).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return patronhttp.NewResponse(fmt.Sprintf("got %s from second HTTP route", rsp.Status)), nil
}

// DoTimingRequest is a helper method to make a request to the seventh example service from other examples
func DoTimingRequest(ctx context.Context) (string, error) {
	request, err := http.NewRequest("GET", "http://localhost:50006/", nil)
	if err != nil {
		return "", fmt.Errorf("failed create route request: %w", err)
	}
	cl, err := clienthttp.New(clienthttp.Timeout(5 * time.Second))
	if err != nil {
		return "", fmt.Errorf("could not create http client: %w", err)
	}

	response, err := cl.Do(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed create get to seventh service: %w", err)
	}

	tb, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to decode timing response body: %w", err)
	}

	var rgx = regexp.MustCompile(`\((.*?)\)`)
	timeInstance := rgx.FindStringSubmatch(string(tb))
	if len(timeInstance) == 1 {
		return "", fmt.Errorf("could not match timeinstance from response %s", string(tb))
	}
	return timeInstance[1], nil
}
