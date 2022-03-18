package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/beatlabs/patron"
	clienthttp "github.com/beatlabs/patron/client/http"
	v2 "github.com/beatlabs/patron/component/http/v2"
	"github.com/beatlabs/patron/component/http/v2/router/httprouter"
	"github.com/beatlabs/patron/encoding/json"
	"github.com/beatlabs/patron/encoding/protobuf"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/log/std"
)

const maxRequests = 1000

var (
	assetsFolder  string
	requestsCount int
	refreshAfter  int64
)

func init() {
	err := os.Setenv("PATRON_JAEGER_SAMPLER_PARAM", "1.0")
	if err != nil {
		fmt.Printf("failed to set sampler env vars: %v", err)
		os.Exit(1)
	}
	// allows running from any folder the 'go run examples/http/main.go'
	var ok bool
	assetsFolder, ok = os.LookupEnv("PATRON_EXAMPLE_ASSETS_FOLDER")
	if !ok {
		assetsFolder = "examples/http/public"
	}

	// allow a thousand requests every 10 seconds
	requestsCount = maxRequests
	refreshAfter = time.Now().Add(10 * time.Second).Unix()
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

	// Set up a simple CORS middleware
	corsMiddleware := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Access-Control-Allow-Origin", "*")
			w.Header().Add("Access-Control-Allow-Methods", "GET, POST")
			w.Header().Add("Access-Control-Allow-Headers", "Origin, Authorization, Content-Type")
			w.Header().Add("Access-Control-Allow-Credentials", "Allow")
			h.ServeHTTP(w, r)
		})
	}

	var routes v2.Routes
	routes.Append(v2.NewGetRoute("/api", getHandler, v2.RateLimiting(50, 50)))
	routes.Append(v2.NewPostRoute("/api", httpHandler))
	routes.Append(httprouter.NewFileServerRoute("/frontend/*path", assetsFolder, assetsFolder+"/index.html"))
	rr, err := routes.Result()
	if err != nil {
		log.Fatalf("failed to create routes: %v", err)
	}

	router, err := httprouter.New(httprouter.Middlewares(corsMiddleware), httprouter.Routes(rr...))
	if err != nil {
		log.Fatalf("failed to create http router: %v", err)
	}

	sig := func() {
		log.Info("exit gracefully...")
		os.Exit(0)
	}

	ctx := context.Background()
	err = service.WithRouter(router).WithSIGHUP(sig).Run(ctx)
	if err != nil {
		log.Fatalf("failed to create and run service %v", err)
	}
}

func getHandler(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write([]byte("Testing Middleware"))
}

func httpHandler(rw http.ResponseWriter, r *http.Request) {
	now := time.Now().Unix()
	if requestsCount <= 0 && refreshAfter > now {
		rw.WriteHeader(http.StatusTooManyRequests)
		rw.Header().Set("Retry-After", strconv.Itoa(int(refreshAfter)))
		_, _ = rw.Write([]byte("no more requests and the refresh time is in the future"))
		return
	}

	requestsCount--
	if refreshAfter <= now {
		requestsCount = maxRequests
	}

	interval, err := DoIntervalRequest()
	if err != nil {
		log.FromContext(r.Context()).Infof("httpHandler: failed to get interval information %v: could it be that the http-cache service is not running ?", err)
	} else {
		log.FromContext(r.Context()).Infof("httpHandler: pipeline initiated at: %s", interval)
	}

	var u examples.User

	err = json.Decode(r.Body, &u)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		_, _ = rw.Write([]byte(fmt.Sprintf("failed to decode request: %v", err)))
		return
	}

	b, err := protobuf.Encode(&u)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(fmt.Sprintf("failed create request: %v", err)))
		return
	}

	httpRequest, err := http.NewRequest("GET", "http://localhost:50001", bytes.NewReader(b))
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(fmt.Sprintf("failed create request: %v", err)))
		return
	}
	httpRequest.Header.Add("Content-Type", protobuf.Type)
	httpRequest.Header.Add("Accept", protobuf.Type)
	httpRequest.Header.Add("Authorization", "Apikey 123456")
	cl, err := clienthttp.New(clienthttp.Timeout(5 * time.Second))
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(fmt.Sprintf("failed execute request: %v", err)))
		return
	}
	rsp, err := cl.Do(httpRequest)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		_, _ = rw.Write([]byte(fmt.Sprintf("failed to perform http request with protobuf payload: %v", err)))
		return
	}
	log.FromContext(r.Context()).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	rw.WriteHeader(http.StatusCreated)
	_, _ = rw.Write([]byte(fmt.Sprintf("got %s from HTTP route", rsp.Status)))
}

// DoIntervalRequest is a helper method to make a request to the http-cache example service from other examples
func DoIntervalRequest() (string, error) {
	request, err := http.NewRequest("GET", "http://localhost:50007/", nil)
	if err != nil {
		return "", fmt.Errorf("failed create route request: %w", err)
	}
	cl, err := clienthttp.New(clienthttp.Timeout(5 * time.Second))
	if err != nil {
		return "", fmt.Errorf("could not create http client: %w", err)
	}

	response, err := cl.Do(request)
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
