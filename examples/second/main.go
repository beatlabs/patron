package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/beatlabs/patron"
	"github.com/beatlabs/patron/examples"
	"github.com/beatlabs/patron/log"
	"github.com/beatlabs/patron/sync"
	patronhttp "github.com/beatlabs/patron/sync/http"
	"github.com/beatlabs/patron/sync/http/auth/apikey"
	tracehttp "github.com/beatlabs/patron/trace/http"
	"github.com/beatlabs/patron/trace/kafka"
	"github.com/pkg/errors"
)

const (
	kafkaTopic  = "patron-topic"
	kafkaBroker = "localhost:9092"
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

func main() {
	name := "second"
	version := "1.0.0"

	err := patron.Setup(name, version)
	if err != nil {
		fmt.Printf("failed to set up logging: %v", err)
		os.Exit(1)
	}

	httpCmp, err := newHTTPComponent(kafkaBroker, kafkaTopic, "http://localhost:50000/second")
	if err != nil {
		log.Fatalf("failed to create processor %v", err)
	}

	auth, err := apikey.New(&apiKeyValidator{validKey: "123456"})
	if err != nil {
		log.Fatalf("failed to create authenticator %v", err)
	}

	// Set up routes
	routes := []patronhttp.Route{
		patronhttp.NewAuthGetRoute("/", httpCmp.second, true, auth),
	}

	srv, err := patron.New(
		name,
		version,
		patron.Routes(routes),
	)
	if err != nil {
		log.Fatalf("failed to create service %v", err)
	}

	err = srv.Run()
	if err != nil {
		log.Fatalf("failed to run service %v", err)
	}
}

type httpComponent struct {
	prd   kafka.Producer
	topic string
}

func newHTTPComponent(kafkaBroker, topic, url string) (*httpComponent, error) {
	prd, err := kafka.NewAsyncProducer([]string{kafkaBroker})
	if err != nil {
		return nil, err
	}
	return &httpComponent{prd: prd, topic: topic}, nil
}

func (hc *httpComponent) second(ctx context.Context, req *sync.Request) (*sync.Response, error) {

	var u examples.User
	err := req.Decode(&u)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode message")
	}

	googleReq, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request for www.google.com")
	}
	cl, err := tracehttp.New(tracehttp.Timeout(5 * time.Second))
	if err != nil {
		return nil, err
	}
	_, err = cl.Do(ctx, googleReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get www.google.com")
	}

	kafkaMsg, err := kafka.NewJSONMessage(hc.topic, &u)
	if err != nil {
		return nil, err
	}

	err = hc.prd.Send(ctx, kafkaMsg)
	if err != nil {
		return nil, err
	}

	log.FromContext(ctx).Infof("request processed: %s %s", u.GetFirstname(), u.GetLastname())
	return nil, nil
}

type apiKeyValidator struct {
	validKey string
}

func (av apiKeyValidator) Validate(key string) (bool, error) {
	if key == av.validKey {
		return true, nil
	}
	return false, nil
}
